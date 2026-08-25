
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

(define planner_direct_presence_probe_preferred? (lambda (probe_rows input_rows)
	(and (number? probe_rows)
		(and (> probe_rows 0)
			(and (number? input_rows)
				(planner_cost_better?
					(planner_direct_presence_probe_cost probe_rows)
					(planner_presence_carrier_cost input_rows probe_rows)))))))

(define stage_direct_probe_cost_preferred? (lambda (stage probe_rows_value)
	(begin
		/* A merged scalar stage exposes several values from the same bounded row.
		Direct lowering performs one probe per requested aggregate, whereas the
		keytable carrier fills all columns in one ordered scan. Cost the complete
		consumer work instead of comparing one probe with one carrier build. */
		(define probe_width (max 1 (count (gs_aggregates stage))))
		(define raw_probe_rows (planner_literal_value probe_rows_value))
		(define probe_rows (if (number? raw_probe_rows)
			(* raw_probe_rows probe_width)
			raw_probe_rows))
		(define input_rows (planner_stage_input_rows (gs_input stage)))
		(define chosen (planner_direct_presence_probe_preferred? probe_rows input_rows))
		(if (and (number? probe_rows) (number? input_rows))
			(planner_guarded_choice chosen
				(list (quote planner_direct_presence_probe_preferred?)
					(list (quote *) probe_rows_value probe_width)
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

/* Return (found expression), rather than the expression alone, because SQL NULL
is itself a valid equality binding. The opposite side must be independent of
the source row; otherwise this is a join equality, not a query-local value. */
(define source_column_equality_binding (lambda (src col condition)
	(reduce
		(split_and_terms (coalesceNil condition true))
		(lambda (found term)
			(if (car found)
				found
				(match term
					'(op left right)
					(if (or (equal? op (quote equal?)) (equal? op (quote equal??)))
						(if (and (equal?? (direct_column_name_for_alias src left) col)
							(not (expr_contains_column_ref? right)))
							(list true right)
							(if (and (equal?? (direct_column_name_for_alias src right) col)
								(not (expr_contains_column_ref? left)))
								(list true left)
								found))
						found)
					_ found)))
		(list false nil))))

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

/* A nested aggregate scan cannot be entered from a filter callback of an
already active scan over the same storage relation. Keep that stage relational
so the consuming join node can prepare/probe its carrier outside the callback. */
(define scalar_probe_input_overlaps_sources? (lambda (stage sources)
	(begin
		(define input (gs_input stage))
		(if (not (query_block? input))
			false
			(reduce (qb_sources input) (lambda (overlaps inner_src)
				(or overlaps
					(and (source_is_base_table? inner_src)
						(reduce (coalesceNil sources '()) (lambda (found outer_src)
							(or found
								(and (source_is_base_table? outer_src)
									(and (equal? (source_schema inner_src) (source_schema outer_src))
										(equal? (source_relation inner_src) (source_relation outer_src))))))
							false))))
				false)))))

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
				(not (scalar_probe_input_overlaps_sources? stage probe_sources))
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
						(stage_lookup_keys_resolve_in_sources? stage probe_sources default_alias))))))))

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

/* A physical probe is a lookup from a relational driver row. Consuming the
only partitioned FROM source would erase the block's row multiplicity
(including empty-input semantics) and leave grouped lowering without a driver. */
(define probe_sources_with_relational_driver (lambda (sources probe_sources relational_sources)
	(if (and (not (empty_list? sources))
		(and (single_source? sources)
			(and (not (empty_list? relational_sources))
				(equal? (count sources) (count probe_sources)))))
		'()
		probe_sources)))

(define partitioned_stage_output_sources (lambda (stages sources)
	(filter (coalesceNil sources '()) (lambda (src)
		(if (not (stage_output_relation? (source_relation src)))
			false
			(begin
				(define stage (stage_by_id stages
					(stage_output_relation_id (source_relation src))))
				(and (not (nil? stage))
					(not (empty_list? (qassoc_get (gs_facts stage) (quote partition_by) '()))))))))))

(define grouped_probe_consumer? (lambda (block)
	(or (not (empty_list? (qb_group block)))
		(or (not (nil? (qb_having block)))
			(query_block_has_aggregates? block)))))

(define presence_probe_output_sources (lambda (stages sources default_alias allow_unbounded)
	(filter (coalesceNil sources '()) (lambda (src)
		(if (not (presence_stage_output_source? stages src))
			false
			(begin
				(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
				(define probe_sources (filter sources (lambda (candidate) (not (equal? (source_alias candidate) (source_alias src))))))
				(and
					(or allow_unbounded
						(presence_stage_probe_allowed_in_context? stage probe_sources))
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
		(define selected_probe_sources (if (nil? retained_order_alias)
			probe_candidates
			(merge_unique (list probe_candidates (list (nth order_lookup 0))))))
		(define probe_sources
			(probe_sources_with_relational_driver sources selected_probe_sources
				(if (and (single_source? sources)
					(and (single_source? selected_probe_sources) (grouped_probe_consumer? block)))
					(partitioned_stage_output_sources stages selected_probe_sources)
					'())))
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
		(define selected_probe_sources
			(probe_sources_with_relational_driver sources probe_sources
				(if (and (single_source? sources)
					(and (single_source? probe_sources) (grouped_probe_consumer? block)))
					(partitioned_stage_output_sources stages probe_sources)
					'())))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias) (if (empty_list? sources) nil (source_alias (car sources)))))
		(define rewritten_sources (rewrite_scalar_first_probe_sources_using
			stages sources selected_probe_sources default_alias))
		(make_query_block
			(qb_schema block)
			/* probe_sources is the caller's already costed and semantics-checked
			selection. Do not rerun the unbounded cache heuristic while removing it. */
			(sources_without_probe_outputs rewritten_sources selected_probe_sources)
			(rewrite_scalar_first_probe_fields stages selected_probe_sources default_alias (qb_fields block))
			(rewrite_scalar_first_probe_expr stages selected_probe_sources default_alias (qb_where block))
			(qb_group block)
			(rewrite_scalar_first_probe_expr stages selected_probe_sources default_alias (qb_having block))
			(rewrite_scalar_first_probe_order stages selected_probe_sources default_alias (qb_order block))
			(qb_limit block)
			(qb_offset block)
			(rewrite_scalar_first_probe_fields stages selected_probe_sources default_alias (qb_hidden block))
			(qb_stages block)
			(join_optimizer_facts_without_aliases
				(qassoc_set
					(qb_facts block)
					(quote consumed_presence_probe_stage_ids)
					(merge_unique (list
						(qassoc_get (qb_facts block) (quote consumed_presence_probe_stage_ids) '())
						(stage_output_source_ids selected_probe_sources))))
				(map selected_probe_sources source_alias))))))

(define query_block_with_presence_probes_using (lambda (stages block allow_unbounded)
	(begin
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias) (if (empty_list? sources) nil (source_alias (car sources)))))
		(query_block_with_presence_probe_sources_using
			stages
			(presence_probe_output_sources stages sources default_alias allow_unbounded)
			block))))

(define query_block_without_stages_after_prepare_using (lambda (stages block)
	(begin
		(define available_stages (if (lowering_catalog? stages)
			(lowering_catalog_stages stages)
			(unique_stages_by_id (merge (list stages (qb_stages block))))))
		(define stage_lookup (if (lowering_catalog? stages) stages available_stages))
		(define rewritten (query_block_with_scalar_first_probes_using stage_lookup block))
		/* Membership has one physical-choice owner, before preparation. Repeating
		that choice here can turn nested scalar dependencies into a second carrier
		graph. Predicate placement is synchronized when the original choice is made,
		so any surviving canonical marker is an invariant violation. */
		(if (expr_contains_membership_truth? (qb_where rewritten))
			(neumann_fail "build_queryplan"
				"logical membership marker survived physical choice") true)
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

(define query_block_with_probe_markers_using_graph (lambda (stages dependency_graph block allow_unbounded_presence)
	(begin
		(define available_stages (if (lowering_catalog? stages)
			(lowering_catalog_stages stages)
			(unique_stages_by_id (merge (list stages (qb_stages block))))))
		(define stage_lookup (if (lowering_catalog? stages) stages available_stages))
		(define scalar_rewritten
			(query_block_with_scalar_first_probes_using_graph stage_lookup dependency_graph block))
		(define membership_rewritten
			(query_block_with_physical_membership_using stage_lookup scalar_rewritten))
		(if (expr_contains_membership_truth? (qb_where membership_rewritten))
			(neumann_fail "build_queryplan"
				"logical membership marker survived physical choice") true)
		(define sources (qb_sources membership_rewritten))
		(define default_alias (qassoc_get (qb_facts membership_rewritten) (quote default_alias) (if (empty_list? sources) nil (source_alias (car sources)))))
		(define presence_probe_sources (presence_probe_output_sources
			stage_lookup sources default_alias allow_unbounded_presence))
		(define rewritten (query_block_with_presence_probes_using
			stage_lookup membership_rewritten allow_unbounded_presence))
		(make_query_block
			(qb_schema rewritten)
			(qb_sources rewritten)
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

(define query_block_with_prepared_sources_using_graph (lambda (stages dependency_graph block)
	(begin
		(define rewritten
			(query_block_with_probe_markers_using_graph stages dependency_graph block false))
		(define stage_lookup (if (lowering_catalog? stages)
			stages
			(query_block_stage_catalog rewritten)))
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
			(qb_stages rewritten)
			(qb_facts rewritten)))))

(define query_block_with_prepared_sources_using (lambda (stages block)
	(query_block_with_prepared_sources_using_graph
		stages (stage_dependency_graph stages) block)))

(define group_stage_final_block (lambda (stage extra_sources)
	(begin
		(define src (gs_input stage))
		(define alias (group_stage_input_alias stage))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		/* A relational scalar-value stage already carries its own empty/cardinality
		semantics. Its ordered aggregates must remain one combined point recipe; a
		synthetic presence column would split that recipe into per-column scans. */
		(define needs_count_filter (and
			(not (scalar_value_stage? stage))
			(and (not (equal? keys '(1))) (not (equal? condition true)))))
		(define ags (if needs_count_filter
			(dedupe_aggregates_by_col (merge (list (gs_aggregates stage) (list aggregate_count_descriptor))))
			(gs_aggregates stage)))
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
				(stable_structural_hash (list
					(group_cache_schema cache)
					(group_cache_relation cache)) true))))))

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
				(map (gs_order stage) (lambda (item) (stable_structural_hash item true)))
				(if (source_is_base_table? (gs_input stage))
					nil
					(list
						(qassoc_get (gs_facts stage) (quote null_semantics) nil)
						(qassoc_get (gs_facts stage) (quote partition_by) '())
						(qassoc_get (gs_facts stage) (quote partition_limit) nil)
						(qassoc_get (gs_facts stage) (quote on_overflow) nil)))))
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
		(lower_stage_prepare_using all_stages lookup_stages stage true)))))

(define lower_unique_stage_prepares_with_graph (lambda (dependency_graph stage_lookup stages)
	(lower_unique_stage_prepares stages (lambda (stage)
		(lower_stage_prepare_using
			(stage_dependency_closure_using_graph dependency_graph stage)
			stage_lookup
			stage
			true)))))

(define lower_presence_stage_prepares_with_graph (lambda (dependency_graph stage_lookup stages)
	(begin
		/* Pure presence chains have complete logical dependency edges. Emit their
		closure once, dependency-first; every local body then consumes the already
		prepared relational output instead of copying the remaining probe suffix. */
		(define ordered (reverse (stage_dependency_closure_many_using_graph
			dependency_graph (collect_stage_prepares stages))))
		(define plans (lower_unique_stage_prepares ordered (lambda (stage)
			(lower_stage_prepare_using
				(stage_dependency_closure_using_graph dependency_graph stage)
				stage_lookup
				stage
				false))))
		(if (empty_list? plans) '() (list (cons (quote !begin) plans))))))

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

(define boolean_recset_domain_source (lambda (input keys)
	(if (not (query_block? input))
		nil
		(reduce (qb_sources input) (lambda (found candidate)
			(if (not (nil? found))
				found
				(if (not (source_is_base_table? candidate))
					nil
					(begin
						(define cols (map keys (lambda (key)
							(direct_column_name_for_alias candidate key))))
						(if (and (not (empty_list? cols))
							(and (not (reduce cols (lambda (missing col)
								(or missing (nil? col))) false))
								(planner_columns_unique? candidate cols)))
							candidate
							nil))))) nil))))

/* A decorrelated stage input is represented as a LEFT-JOIN chain, including
the domain source because it was nullable relative to the former outer row.
When the stage is evaluated as an independent relation, that first domain is
the driver and must be scanned normally; only its dependent sources remain
outer joins. */
(define query_block_with_inner_domain_source (lambda (input domain_src)
	(make_query_block
		(qb_schema input)
		(map (qb_sources input) (lambda (src)
			(if (equal? (source_alias src) (source_alias domain_src))
				(list (source_alias src) (source_schema src) (source_relation src)
					false (source_join_expr src))
				src)))
		(qb_fields input) (qb_where input) (qb_group input) (qb_having input)
		(qb_order input) (qb_limit input) (qb_offset input) (qb_hidden input)
		(qb_stages input)
		(filter (qb_facts input) (lambda (entry)
			(match entry
				(cons key _value) (not (equal? key (quote join_plan)))
				_ true))))))

(define lower_group_stage_prepare_using (lambda (all_stages lookup_stages stage include_nested_prepares result_sink)
	(begin
		(define src (gs_input stage))
		(define prepare_catalog (unique_stages_by_id (merge (list (list stage) all_stages))))
		(define prepare_dependency_graph (stage_dependency_graph prepare_catalog))
		(define prepare_dependencies
			(stage_dependency_closure_using_graph prepare_dependency_graph stage))
		(define relational_presence_chain (and (> (count prepare_dependencies) 2)
			(reduce prepare_dependencies (lambda (eligible dependency)
				(and eligible (and (group_stage? dependency)
					(presence_probe_stage? dependency)))) true)))
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
		/* A boolean RecSet producer is also executed by cardinality observations,
		before the main query's lexical bindings exist. Keep invariant probes as
		query-scoped markers inside that closed producer; its own lowering memoizes
		them once. Other result sinks retain the established outer binding. */
		(define closed_boolean_sink (equal? result_sink (quote boolean-recset)))
		(define stage_lookup (if closed_boolean_sink
			(stage_lookup_with_inline_query_invariant_probes raw_stage_lookup)
			(stage_lookup_with_query_invariant_probe_bindings
				raw_stage_lookup invariant_probe_entries)))
		(define invariant_probe_bindings (if closed_boolean_sink '()
			(query_invariant_probe_bindings invariant_probe_entries)))
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
		/* A nested presence chain represented as correlated scalar probes copies
		the complete remaining probe suffix into every parent prepare. Its stage
		caches are already dependency-ordered here, so consume those relational
		outputs directly once the chain contains more than one dependency. */
		(define rewritten_src (if (query_block? membership_src)
			(if (equal? result_sink (quote boolean-recset))
				(query_block_with_probe_markers_using_graph
					stage_lookup prepare_dependency_graph membership_src true)
				(if relational_presence_chain
					membership_src
					(query_block_with_presence_probes_using stage_lookup membership_src false)))
			membership_src))
		(define rewrite_sources (if (query_block? membership_src) (qb_sources membership_src) '()))
		(define rewrite_default_alias (if (query_block? src)
			(qassoc_get (qb_facts membership_src) (quote default_alias) (if (empty_list? rewrite_sources) nil (source_alias (car rewrite_sources))))
			nil))
		(define presence_probe_sources_for_rewrite (if (query_block? src)
			(presence_probe_output_sources stage_lookup rewrite_sources rewrite_default_alias
				(equal? result_sink (quote boolean-recset)))
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
		(define condition (if (query_block? src)
			(rewrite_scalar_first_probe_expr stage_lookup presence_probe_sources_for_rewrite rewrite_default_alias (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
			(coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true)))
		(define needs_count_filter (and
			(not (scalar_value_stage? stage))
			(and (not (equal? keys '(1))) (not (equal? condition true)))))
		(define ags (if needs_count_filter
			(dedupe_aggregates_by_col (merge (list (gs_aggregates stage) (list aggregate_count_descriptor))))
			(gs_aggregates stage)))
		(define lowering_ags (if (query_block? src)
			(rewrite_scalar_first_probe_aggregates stage_lookup presence_probe_sources_for_rewrite rewrite_default_alias ags)
			ags))
		(define result_sink_ags (if (and (equal? result_sink (quote boolean-recset))
			(query_block? src))
			(rewrite_scalar_first_probe_aggregates stage_lookup
				(filter rewrite_sources (lambda (candidate)
					(stage_output_relation? (source_relation candidate))))
				rewrite_default_alias ags)
			lowering_ags))
		(define key_names (group_key_cols keys))
		(define aggregate_condition (replace_group_session_expr stage keys key_names condition))
		(define grouptbl (group_cache_relation cache))
		(define initializer_owner (qassoc_get (gs_facts stage) (quote keytable_initializer_owner) true))
		(define scalar_single_stage (scalar_value_stage? stage))
		(define scalar_query_stage (and (query_block? src)
			(and scalar_single_stage
				(and (equal? (qassoc_get (gs_facts stage) (quote partition_limit) nil) 2)
					(and (equal? (qassoc_get (gs_facts stage) (quote on_overflow) nil) (quote error))
						(and (equal? (count ags) 2)
							(equal? (cadr ags) aggregate_count_descriptor)))))))
		(define scalar_order_base_stage (and (not query_input)
			(compatible_scalar_order_aggregates? ags)))
		/* Aggregate column names belong to the immutable logical stage. Prepared
		input and rewritten descriptors below affect execution only. Base aggregate
		columns use their direct physical builder and need no canonical list here. */
		(define aggregate_cols (if (or query_input scalar_order_base_stage)
			(map ags (lambda (ag) (aggregate_col_name_using src ag)))
			'()))
		(define scalar_aggregate_stage (scalar_aggregate_probe_stage? stage))
		(define prepared_src (if (query_block? rewritten_src)
			(if (equal? result_sink (quote boolean-recset))
				rewritten_src
				(if scalar_aggregate_stage
					(begin
						(define constant_reorder_stages (if (lowering_catalog? stage_lookup)
							stage_lookup
							(unique_stages_by_id (merge (list stage_lookup (qb_stages rewritten_src))))))
						(if (not (empty_list? (filter (qb_sources rewritten_src) (lambda (src)
							(constant_scalar_or_presence_stage_output_source? constant_reorder_stages src)))))
							(query_block_without_stages_after_eager_prepare_with_constant_scalars_first constant_reorder_stages rewritten_src)
							(query_block_without_stages_after_eager_prepare_using stage_lookup rewritten_src)))
					(query_block_without_stages_after_eager_prepare_using stage_lookup rewritten_src)))
			rewritten_src))
		(define direct_nested_stages (if (query_block? rewritten_src)
			(merge_unique (list
				(query_block_stages_to_prepare_using stage_lookup rewritten_src)
				(available_stage_outputs_from_sources_using stage_lookup (qb_sources rewritten_src))
				(available_stage_outputs_from_sources_using stage_lookup (group_stage_final_extra_source_refs stage))
				(group_cache_stages_from_sources stage_lookup (qb_sources rewritten_src))
				(group_cache_stages_from_sources stage_lookup (group_stage_final_extra_source_refs stage))
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
		(define nested_prepare (if (and include_nested_prepares (query_block? rewritten_src))
			(if relational_presence_chain
				(lower_presence_stage_prepares_with_graph
					prepare_dependency_graph stage_lookup nested_stages)
				(lower_unique_stage_prepares_using prepare_catalog stage_lookup nested_stages))
			'()))
		(define nested_materialize (if (and include_nested_prepares (query_block? rewritten_src))
			(lower_stage_materialize_all nested_stages) '()))
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
		(define boolean_row_keys (if (equal? result_sink (quote boolean-recset))
			(scalar_first_probe_recset_row_keys stage
				(scalar_first_probe_carrier_source prepared_src)) '()))
		(define boolean_domain_src (if (equal? result_sink (quote boolean-recset))
			(boolean_recset_domain_source prepared_src boolean_row_keys) nil))
		(define boolean_input (if (nil? boolean_domain_src) prepared_src
			(query_block_with_inner_domain_source prepared_src boolean_domain_src)))
		(if (and (equal? result_sink (quote boolean-recset)) (nil? boolean_domain_src))
			(neumann_fail "build_queryplan" "boolean RecSet sink requires a unique base-table domain")
			true)
		(define lowered_plan_core (if (equal? result_sink (quote boolean-recset))
			(build_query_boolean_recset_plan
				stage_lookup boolean_input boolean_domain_src boolean_row_keys result_sink_ags)
			(if scalar_query_stage
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
								aggregate_prepare_expr)))))))
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

(define lower_stage_prepare_using (lambda (all_stages lookup_stages stage include_nested_prepares)
	(if (group_stage? stage)
		(lower_group_stage_prepare_using all_stages lookup_stages stage include_nested_prepares nil)
		(if (orc_stage? stage)
			(lower_orc_stage_prepare stage)
			(if (window_stage? stage)
				(lower_window_stage_prepare stage)
				(neumann_fail "build_queryplan" "unknown logical stage"))))))
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

(define stage_catalog_entries (lambda (stages)
	(if (lowering_catalog? stages)
		(lowering_catalog_stages stages)
		(merge (map (coalesceNil stages '()) (lambda (stage)
			(if (lowering_catalog? stage)
				(lowering_catalog_stages stage)
				(list stage))))))))

(define merge_stage_catalogs (lambda (catalogs)
	(unique_stages_by_id
		(merge (map (coalesceNil catalogs '()) lowering_catalog_stages)))))

(define stage_catalog_with_nested (lambda (stages)
	(unique_stages_by_id
		(merge (map (stage_catalog_entries stages) nested_stage_catalog)))))

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
			(lower_group_stage_prepare_using stage_catalog stage_catalog stage true nil)
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
							(list (lower_group_stage_prepare_using (cons main_stage stage_catalog) main_stage_lookup main_stage true nil))
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

/* A table-function relation is a logical source descriptor. Its physical
lowering streams the produced row lists directly through the ordinary bound
column expressions, without creating a temporary storage table. */
(define table_function_column_index (lambda (columns name index)
	(match columns
		(cons column rest) (if (equal?? column name) index
			(table_function_column_index rest name (+ index 1)))
		_ (neumann_fail "build_queryplan" (concat "unknown table-function column: " name)))))

(define lower_table_function_expr (lambda (src columns row expr)
	(match expr
		((symbol get_column) tblvar _ col _) (if (or (nil? tblvar) (equal?? tblvar (source_alias src)))
			(list (quote nth) row (table_function_column_index columns col 0)) expr)
		((quote get_column) tblvar _ col _) (lower_table_function_expr src columns row
			(list (quote get_column) tblvar false col false))
		(cons head tail) (cons head (map tail (lambda (item) (lower_table_function_expr src columns row item))))
		_ expr)))

(define lower_table_function_query_block (lambda (block src)
	(match (source_relation src)
		((symbol table-function) kind args columns) (begin
			(if (or (not (empty_list? (qb_group block)))
				(or (not (nil? (qb_having block))) (query_block_has_aggregates? block)))
				(neumann_fail "build_queryplan" "aggregate over table function is not implemented") true)
			(define fields (expand_query_block_fields (list src) (qb_fields block)))
			(define row (quote __table_function_row))
			(define condition (lower_table_function_expr src columns row
				(combine_where (qb_where block) (source_join_expr src))))
			(define emit (list (quote resultrow) (cons (quote list)
				(map_assoc fields (lambda (_title expr) (lower_table_function_expr src columns row expr))))))
			(list (quote map)
				(cons (quote pg_json_table_rows) (cons kind args))
				(list (quote lambda) (list row) (list (quote if) condition emit nil))))
		_ (neumann_fail "build_queryplan" "invalid table-function relation"))))

(define lower_query_block_core (lambda (block)
	(if (empty_list? (qb_sources block))
		(lower_zero_source_query_block block)
		(if (single_source? (qb_sources block))
			(if (table_function_relation? (source_relation (car (qb_sources block))))
				(lower_table_function_query_block block (car (qb_sources block)))
				(lower_single_source_query_block block))
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

/* A physical membership marker is the exclusive owner of its carrier. Stop at
the marker instead of walking into the embedded logical stage; cache-backed
alternatives prepare themselves, while raw RecSet and direct-probe alternatives
must not pay for an unused group cache. */
(define physical_membership_probe_stage_ids (lambda (expr)
	(match expr
		((symbol driver_membership_probe) stage _probe) (list (gs_id stage))
		((quote driver_membership_probe) stage _probe) (list (gs_id stage))
		((symbol driver_membership_subscan_probe) stage _probe) (list (gs_id stage))
		((quote driver_membership_subscan_probe) stage _probe) (list (gs_id stage))
		(cons _head tail) (merge_unique (map tail physical_membership_probe_stage_ids))
		_ '())))

(define query_block_stages_to_prepare_base_using (lambda (all_stages block)
	(begin
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias) (if (empty_list? sources) nil (source_alias (car sources)))))
		(define consumed_probe_ids (qassoc_get (qb_facts block) (quote consumed_presence_probe_stage_ids) '()))
		(define consumed_source_probe_ids (stage_output_source_ids (probe_output_sources_for_block
			all_stages sources default_alias (qb_limit block) (qb_where block) (query_block_probe_consumers block))))
		(define stage_output_ids (stage_output_source_ids sources))
		(define physical_membership_stage_ids
			(physical_membership_probe_stage_ids (query_block_probe_consumers block)))
		(define direct (filter (qb_stages block) (lambda (stage)
			(and
				(not (contains? physical_membership_stage_ids (gs_id stage)))
				(and
					(stage_direct_prepare_source_visible? block sources default_alias stage)
					(stage_direct_prepare_semantic_candidate? consumed_probe_ids consumed_source_probe_ids stage_output_ids stage))))))
		direct)))

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
						(lazy_stage_prepare_bindings stage_lookup lazy_stages)
						(prepared_stage_bindings eager_stages)
						(lower_unique_stage_prepares_with_graph eager_dependency_graph eager_stage_lookup eager_stages)
						(lower_stage_materialize_all eager_stages)))
					core_block))))))

(define prepare_simple_query_block_physical_core (lambda (block)
	(prepare_simple_query_block_physical_core_chosen
		(query_block_with_physical_membership_choices
			(query_block_with_physical_membership_using
				(query_block_stage_lookup block) block)))))

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
		(define direct_dependencies
			(coalesceNil (get_assoc dependency_graph (logical_stage_key stage)) '()))
		(list
			(list (quote context) "session")
			(stage_prepare_key stage)
			(list (quote once)
				(list (quote lambda)
					'()
					(list (quote !begin)
						(cons (quote !begin) (map direct_dependencies stage_prepare_call_expr))
						(lower_stage_prepare_using dependencies stage_catalog stage true)
						true)))))))

(define lazy_stage_prepare_bindings (lambda (stages selected)
	(begin
		(define dependency_graph (stage_dependency_graph stages))
		(define lazy_stages (stage_dependency_closure_many_using_graph
			dependency_graph (collect_stage_prepares selected)))
		(map lazy_stages (lambda (stage)
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
	(if (table_function_relation? tbl)
		(table_function_columns tbl)
		(map (get_schema schema tbl) (lambda (col) (col "Field"))))))

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

(define physical_expr_refs_any_alias? (lambda (sources default_alias aliases expr)
	(reduce (coalesceNil aliases '()) (lambda (found alias)
		(or found (not (empty_list?
			(extract_columns_for_join_alias sources default_alias alias expr))))) false)))

(define downstream_sources_preserve_driver_rows? (lambda (sources default_alias final_condition stages)
	(begin
		(define rest_sources (cdr sources))
		/* A driver membership marker embeds its complete logical stage as data.
		Those internal aliases are not downstream row predicates: the marker is
		lowered and evaluated on the driver itself before its native LIMIT. */
		(define driver_membership (driver_membership_for_source (car sources) final_condition))
		(define downstream_condition (strip_driver_membership_for_source
			(car sources) final_condition driver_membership))
		(and
			(reduce rest_sources (lambda (ok src) (and ok (source_outer? src))) true)
			/* A 0:1 nullable lookup preserves cardinality by itself, but a WHERE
			predicate reading its value can still reject the driver row. Only pure
			projection lookups may stay behind the native driver LIMIT. */
			(and (not (physical_expr_refs_any_alias?
				sources default_alias (source_aliases rest_sources) downstream_condition))
				(ordered_join_sources_are_unique_lookups_acc
					sources default_alias stages final_condition (list (car sources)) rest_sources))))))

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
		(order_items_follow_join_tree?
			ordered_sources default_alias order_items stages final_condition))))

(define downstream_sources_at_most_one_driver_row? (lambda (sources default_alias final_condition stages)
	(if (empty_list? sources)
		true
		(ordered_join_sources_are_unique_lookups_acc
			sources default_alias stages final_condition (list (car sources)) (cdr sources)))))

(define driver_limit_cannot_brake? (lambda (sources default_alias final_condition offset_value limit_value stages)
	(begin
		(define compile_offset (planner_literal_value offset_value))
		(define compile_limit (planner_literal_value limit_value))
		(define rows (if (empty_list? sources) nil
			(if (source_unique_point_condition? (car sources) final_condition)
				1
				/* A selectivity estimate is not an upper bound. Native LIMIT may
				stop at the driver only when its complete relation is provably
				inside the window; downstream 0:1 predicates can still reject an
				estimatedly selective driver row. */
				(planner_source_row_count (car sources)))))
		(and (or (nil? compile_offset) (equal? compile_offset 0))
			(and (number? compile_limit)
				(and (>= compile_limit 0)
					(and (number? rows)
						(and (<= rows compile_limit)
							(downstream_sources_at_most_one_driver_row?
								sources default_alias final_condition stages)))))))))

(define ordered_join_limit_requires_complete_rows? (lambda (sources default_alias final_condition offset_value limit_value stages)
	(and
		(query_limit_active? offset_value limit_value)
		(and (not (driver_limit_cannot_brake?
			sources default_alias final_condition offset_value limit_value stages))
			(and (not (empty_list? (cdr sources)))
				(not (downstream_sources_preserve_driver_rows? sources default_alias final_condition stages)))))))

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
	(symbol (concat "__membership_recset_" (stable_structural_hash (list
		(source_schema src)
		(source_relation src)
		(gs_id (nth membership 0))
		(nth membership 1)
		(nth membership 2)) true)))))

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

(define membership_recset_bindings_using (lambda (src memberships consumer driver_rows_override allow_ordered_batch)
	(filter (map memberships (lambda (membership)
		(begin
			(define expr (recset_project_join_expr_for_membership_using
				src membership consumer driver_rows_override allow_ordered_batch))
			(if (nil? expr) nil (list membership (membership_recset_var src membership) expr)))))
		(lambda (binding) (not (nil? binding))))))

(define membership_recset_bindings (lambda (src memberships)
	(membership_recset_bindings_using src memberships nil nil false)))

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

/* Computed membership keys are already stored canonically as k0 in the group
cache. Probing that keytable is cheaper than first copying its rows into a
query-local RecSet index. This selected carrier prepares its cache once before
returning the closure; the closure then performs only the indexed presence
lookup for each ordered driver candidate. */
(define membership_cache_key_probe_expr (lambda (stage)
	(begin
		(define cache (group_stage_cache stage))
		(define probe_var (symbol "__membership_probe"))
		(define key_name "k0")
		(define value_col (aggregate_col_name_using (gs_input stage) (car (gs_aggregates stage))))
		(define filtercols (list key_name value_col))
		(prepared_group_cache_expr stage
			(list (quote lambda) (list probe_var)
				(list (quote scan_exists)
					'(session "__memcp_tx")
					(list (quote table) (group_cache_schema cache) (group_cache_relation cache))
					(cons (quote list) filtercols)
					(list (quote lambda) (map filtercols symbol)
						(list (quote and)
							(list (quote equal??) (symbol key_name)
								(list (quote outer) probe_var))
							(stage_recset_value_filter_term stage value_col)))))))))

(define membership_keyset_parts (lambda (membership)
	(begin
		(define descriptor (membership_keyset_descriptor membership))
		(if (nil? descriptor)
			(begin
				(define stage (nth membership 0))
				(define input (gs_input stage))
				(if (not (union_block? input))
					(if (and (group_stage? stage)
						(equal? (count (gs_keys stage)) 1))
						(begin
							(define cache (group_stage_cache stage))
							(list
								(list (quote table)
									(group_cache_schema cache) (group_cache_relation cache))
								"k0"
								(membership_cache_key_probe_expr stage)
								(quote group_cache_probe)))
						nil)
					(begin
						(define carriers (map (union_branches input) candidate_recset_branch_carrier))
						(define supported (and (not (empty_list? carriers))
							(not (reduce carriers (lambda (missing carrier)
								(or missing (nil? carrier))) false))))
						(if (not supported)
							nil
							(begin
								(define first_carrier (car carriers))
								(define compatible (reduce (cdr carriers) (lambda (same carrier)
									(and same
										(and (equal? (source_table_expr (nth carrier 0))
											(source_table_expr (nth first_carrier 0)))
											(equal? (nth carrier 1) (nth first_carrier 1))))) true))
								(if (not compatible)
									nil
									(list
										(source_table_expr (nth first_carrier 0))
										(nth first_carrier 1)
										(list (quote recset_union)
											(cons (quote list) (map carriers (lambda (carrier)
												(nth carrier 2))))))))))))
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
		(if (not supported)
			'()
			(if (not compatible)
				(merge (map memberships (lambda (membership)
					(membership_keyset_bindings (list membership)))))
				(begin
					(define keyset_var (symbol (concat "__membership_keyset_"
						(stable_structural_hash (map memberships (lambda (membership)
							(gs_id (nth membership 0)))) true))))
					(define group_cache_probe (and (single_source? parts)
						(equal? (count (car parts)) 4)))
					(define candidate_recsets (map parts (lambda (item) (nth item 2))))
					(define candidate_recset (if (single_source? candidate_recsets)
						(car candidate_recsets)
						(list (quote recset_union) (cons (quote list) candidate_recsets))))
					(define keyset_expr (if group_cache_probe
						(nth (car parts) 2)
						(list (quote recset_key_index)
							'(session "__memcp_tx")
							candidate_recset
							(quoted_runtime_list (list (nth (car parts) 1))))))
					(map memberships (lambda (membership)
						(list membership keyset_var keyset_expr)))))))))

(define unique_membership_keyset_bindings (lambda (bindings)
	(reduce bindings (lambda (unique binding)
		(if (reduce unique (lambda (found existing)
			(or found (equal? (nth existing 1) (nth binding 1)))) false)
			unique
			(cons binding unique))) '())))

(define wrap_membership_keyset_bindings (lambda (bindings body)
	(if (empty_list? bindings)
		body
		(begin
			(define unique (reverse (unique_membership_keyset_bindings bindings)))
			(cons
				(list (quote lambda) (map unique (lambda (binding) (nth binding 1))) body)
				(map unique (lambda (binding) (nth binding 2))))))))

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

(define ordered_batch_membership_terms (lambda (src condition)
	(filter (map (split_and_terms (coalesceNil condition true)) (lambda (term)
		(begin
			(define marker (driver_membership_probe_term term))
			(if (nil? marker)
				nil
				(begin
					(define target_col (direct_column_name_for_alias src (nth marker 1)))
					(if (nil? target_col)
						nil
						(list (nth marker 0) (nth marker 1) target_col term)))))))
		(lambda (membership) (not (nil? membership))))))

(define scan_input_recset_for_condition (lambda (sources default_alias src input condition probe_work_rows)
	(if (equal? condition true)
		input
		(begin
			(define cols (extract_columns_for_alias src condition))
			(list (quote scan_recset)
				'(session "__memcp_tx")
				input
				(cons (quote list) cols)
				(list (quote lambda)
					(map cols (lambda (col)
						(scan_callback_symbol_for_alias (source_alias src) col)))
					(lower_column_expr_for_join_truth_context
						sources default_alias condition probe_work_rows)))))))

/* Estimate the rows which reach the residual predicate from the exact same
membership facts used to choose the outer carrier. The residual lowerer can
therefore choose RecSet, keytable, or direct probes for the smaller batch just
as it would for an ordinary scan input. */
(define batch_membership_survivor_rows (lambda (memberships probe_work_rows)
	(reduce memberships (lambda (rows membership)
		(if (not (number? (planner_literal_value rows)))
			rows
			(begin
				(define stage (nth membership 0))
				(define facts (merge (list
					(membership_candidate_work_facts stage)
					(gs_facts stage))))
				(define candidate_input_rows (coalesceNil
					(qassoc_get facts (quote membership_candidate_input_rows) nil)
					(planner_stage_input_rows (gs_input stage))))
				(define candidate_rows (qassoc_get facts
					(quote membership_candidate_estimated_rows) candidate_input_rows))
				(if (or (not (number? candidate_input_rows))
					(not (number? candidate_rows)))
					rows
					(begin
						/* For an ordered LIMIT, rows is the requested survivor count,
						not the number of driver rows visited. Use the same adaptive
						visit estimate as ordered_batch_accept_cost before applying the
						candidate density. Otherwise a 10%-dense candidate for LIMIT 72
						would incorrectly lower the residual as seven probes even though
						the batch expands to roughly 720 driver rows and 72 survivors. */
						(define visited_rows (membership_expected_driver_rows_visited
							candidate_input_rows candidate_rows
							(planner_literal_value rows) facts))
						(* visited_rows
							(membership_candidate_density
								candidate_input_rows candidate_rows facts)))))))
		probe_work_rows)))

/* Compile one exact predicate pipeline over an arbitrary RecSet input. The
ordered_batch_accept alternative specifically represents candidate-first
execution; the separately costed prefiltered_candidate_keyset alternative owns
driver-predicate-first execution. Keeping those alternatives distinct makes
the emitted work agree with the cost comparison. All residual conditions then
re-enter the ordinary expression lowerer with the candidate-reduced row count,
so complex ACL trees receive the same per-node physical choices as any scan. */
(define ordered_batch_filter_expr (lambda (sources default_alias src condition memberships probe_work_rows acceptance_cols acceptance_probe)
	(begin
		(define input_batch (symbol "__ordered_input_batch"))
		(define residual (reduce memberships (lambda (remaining membership)
			(strip_driver_membership_for_source src remaining membership)) condition))
		(define membership_batch (symbol "__ordered_membership_batch"))
		(define late_batch (symbol "__ordered_late_batch"))
		(define membership_exprs (map memberships (lambda (membership)
			(batch_membership_expr src membership input_batch))))
		(if (reduce membership_exprs (lambda (unsupported expr)
			(or unsupported (nil? expr))) false)
			nil
			(begin
				(define membership_expr (if (empty_list? membership_exprs)
					input_batch
					(list (quote recset_intersect)
						(cons (quote list) (cons input_batch membership_exprs)))))
				(define residual_probe_work_rows
					(batch_membership_survivor_rows memberships probe_work_rows))
				(planner_record_physical_decision (list
					(list "decision" "batch_predicate_lowering")
					(list "chosen" "candidate_then_residual")
					(list "reason" "selected_ordered_batch_carrier_cost")
					(list "inputs" (list
						(list "input_rows" (planner_literal_value probe_work_rows))
						(list "residual_probe_rows"
							(planner_literal_value residual_probe_work_rows))
						(list "membership_count" (count memberships))
						(list "residual_probe_count" (count (expr_probe_stages residual)))))))
				(define late_expr (scan_input_recset_for_condition
					sources default_alias src membership_batch residual
					residual_probe_work_rows))
				(define accepted_expr (if (equal? acceptance_probe true)
					late_batch
					(list (quote scan_recset)
						'(session "__memcp_tx")
						late_batch
						(cons (quote list) acceptance_cols)
						(list (quote lambda)
							(map acceptance_cols (lambda (col)
								(scan_callback_symbol_for_alias (source_alias src) col)))
							acceptance_probe))))
				(list (quote lambda) (list input_batch)
					(list
						(list (quote lambda) (list membership_batch)
							(list
								(list (quote lambda) (list late_batch) accepted_expr)
								late_expr))
						membership_expr)))))))

/* Produce the exact driver subset shared by a prefiltered membership tree.
Every remaining top-level conjunct is evaluated once while building the driver
RecSet; membership edges retain their own physical operators. */
(define prefiltered_driver_recset_expr_for_membership (lambda (src source_table condition membership)
	(begin
		(define residual_condition (strip_driver_membership_for_source src condition membership))
		(define driver_condition (combine_where_terms
			(filter (split_and_terms (coalesceNil residual_condition true)) (lambda (term)
				(not (expr_contains_driver_membership? term))))
			true))
		(if (equal? driver_condition true)
			nil
			(begin
				(define alias (source_alias src))
				(define cols (extract_columns_for_alias src driver_condition))
				(list (quote scan_recset)
					'(session "__memcp_tx")
					source_table
					(cons (quote list) cols)
					(list (quote lambda)
						(map cols (lambda (col) (scan_callback_symbol_for_alias alias col)))
						(lower_column_expr_for_alias src driver_condition))))))))

(define ordered_batch_count_estimate (lambda (first_batch visited_rows)
	(if (or (not (number? first_batch)) (or (<= first_batch 0) (not (number? visited_rows))))
		1
		(if (<= visited_rows first_batch)
			1
			(+ 1 (ordered_batch_count_estimate (* first_batch 2) (- visited_rows first_batch)))))))

(define ordered_batch_accept_cost (lambda (facts)
	(begin
		(define candidate_input_rows (qassoc_get facts (quote membership_candidate_input_rows) nil))
		/* Physical consumers use the one cardinality produced by logical costing.
		They may compare operators, but must not reinterpret statistics. */
		(define candidate_rows (qassoc_get facts
			(quote membership_candidate_estimated_rows) candidate_input_rows))
		(define driver_rows (qassoc_get facts (quote membership_driver_rows) nil))
		(define driver_input_rows (membership_driver_input_rows driver_rows facts))
		(define visited_rows (membership_expected_driver_rows_visited
			candidate_input_rows candidate_rows driver_rows facts))
		(define first_batch (+
			(coalesceNil (qassoc_get facts (quote membership_order_offset) nil) 0)
			(coalesceNil (qassoc_get facts (quote membership_order_limit) nil) 0)))
		(define batches (ordered_batch_count_estimate first_batch visited_rows))
		(define probe_branches (max 1
			(qassoc_get facts (quote membership_candidate_probe_branches) 1)))
		(define candidate_scan_invocations (max 1
			(qassoc_get facts (quote membership_candidate_scan_invocations) probe_branches)))
		(define driver_scan_invocations (max 1
			(qassoc_get facts (quote membership_driver_scan_invocations) 1)))
		/* scan_order_batch_accept requests disjoint windows, but the current
		ordered primitive has no resumable cursor. Every window therefore pays
		for another ordered traversal of the driver, irrespective of how few
		rows that window returns. Keep this explicit so calibration can price
		the implementation that is actually emitted. */
		/* Reuse the calibrated scan_order work unit. Without a resumable cursor,
		each batch repeats the same quadratic ordered traversal; a separate linear
		coefficient would describe the same operator inconsistently. */
		(define ordered_driver_work_units (* batches
			(/ (* driver_input_rows driver_input_rows) 1000000)))
		(define candidate_fraction (if (and (number? driver_input_rows) (> driver_input_rows 0))
			(min 1 (/ visited_rows driver_input_rows)) 1))
		/* Disjoint driver batches may contain the same foreign-key value. In the
		absence of key-frequency statistics, charge every batch for its projected
		fraction instead of assuming candidate source rows are globally disjoint. */
		(define candidate_repeat_fraction (min batches (* candidate_fraction batches)))
		(define candidate_work_rows (* candidate_input_rows candidate_repeat_fraction))
		(define candidate_match_rows (* candidate_rows candidate_repeat_fraction))
		(define projection_rows (* probe_branches
			(+ (* visited_rows 2) candidate_work_rows candidate_match_rows)))
		(define broad_text_rows (*
			(qassoc_get facts (quote membership_candidate_broad_text_match_rows) 0)
			candidate_repeat_fraction))
		(define broad_text_bytes (*
			(qassoc_get facts (quote membership_candidate_broad_text_match_bytes) 0)
			candidate_repeat_fraction))
		(planner_cost_add (planner_cost
			(+ (* (+ driver_scan_invocations (* batches candidate_scan_invocations))
				planner_membership_scan_invocation_ns)
				(* batches driver_scan_invocations
					planner_membership_ordered_scan_invocation_ns))
			(+
				(* ordered_driver_work_units planner_membership_ordered_driver_input_row_ns)
				(* (+ visited_rows candidate_work_rows projection_rows)
					planner_membership_scan_row_ns)
				(* candidate_work_rows
					(qassoc_get facts (quote membership_candidate_filter_columns) 0)
					planner_membership_filter_column_row_ns)
				(* candidate_work_rows
					(qassoc_get facts (quote membership_candidate_expression_operations) 0)
					planner_membership_expression_operation_row_ns)
				(* broad_text_rows planner_membership_broad_text_match_row_ns)
				(* broad_text_bytes planner_membership_broad_text_match_byte_ns))
			0 0 0
			(* projection_rows
				planner_membership_recset_build_row_ns)
			(* projection_rows 8)
			0 visited_rows 0.55)
			(planner_membership_direct_probe_cost
				(* visited_rows
					(membership_candidate_density candidate_input_rows candidate_rows facts)
					(qassoc_get facts (quote membership_downstream_probe_branches) 0)))
			visited_rows 0.55))))

(define membership_row_number_consumer? (lambda (membership direct_order_limit)
	(and (not (nil? membership))
		(and (not direct_order_limit)
			(equal? (qassoc_get (gs_facts (nth membership 0))
				(quote membership_consumer) nil)
				(quote order_limit))))))

/* An exact RecSet used by ORDER/LIMIT has two physical consumers. Scanning the
RecSet directly is proportional to its cardinality. Scanning the ordered base
table and probing membership is proportional to the expected prefix needed to
fill the requested window. Neither dominates for all densities, so this is a
cost decision with an exact request-local observation and a crossover guard. */
(define ordered_recset_expected_base_rows (lambda (source_rows carrier_rows window_rows)
	(if (or (not (number? source_rows)) (<= source_rows 0))
		window_rows
		(if (or (not (number? carrier_rows)) (<= carrier_rows 0))
			source_rows
			(min source_rows (max window_rows
				(/ (* window_rows source_rows) carrier_rows)))))))

(define ordered_direct_recset_cost (lambda (carrier_rows)
	(planner_cost
		planner_membership_ordered_scan_invocation_ns
		(+
			(* carrier_rows planner_membership_scan_row_ns)
			(* (/ (* carrier_rows carrier_rows) 1000000)
				planner_membership_ordered_driver_input_row_ns))
		0 0 0 0 0 0 carrier_rows 0.95)))

(define ordered_base_membership_cost (lambda (source_rows carrier_rows window_rows)
	(begin
		(define visited_rows
			(ordered_recset_expected_base_rows source_rows carrier_rows window_rows))
		(planner_cost
			planner_membership_ordered_scan_invocation_ns
			(* visited_rows (+ planner_membership_scan_row_ns
				planner_membership_recset_probe_row_ns))
			0 0 0 0 0 0 visited_rows 0.95))))

(define ordered_recset_crossover_search (lambda (source_rows window_rows low high remaining)
	(if (or (<= remaining 0) (>= low high))
		low
		(begin
			(define mid (/ (+ low high) 2))
			(if (planner_cost_better?
				(ordered_direct_recset_cost mid)
				(ordered_base_membership_cost source_rows mid window_rows))
				(ordered_recset_crossover_search source_rows window_rows (+ mid 1) high (- remaining 1))
				(ordered_recset_crossover_search source_rows window_rows low mid (- remaining 1)))))))

/* Preparations execute immediately before the compiled plan and therefore do
not live inside with_physical_query_context's lexical bindings. Rebind only the
three physical context symbols; quoted data remains opaque. */
(define ordered_recset_observation_expr (lambda (expr)
	(if (equal? expr (physical_query_session_symbol))
		(list (quote context) "session")
		(if (equal? expr (physical_query_scope_symbol))
			(list (quote context) "query")
			(if (equal? expr (physical_query_tx_symbol))
				(list (list (quote context) "session") "__memcp_tx")
				(match expr
					((symbol quote) _value) expr
					((quote quote) _value) expr
					(cons head tail) (cons
						(ordered_recset_observation_expr head)
						(map tail ordered_recset_observation_expr))
					_ expr))))))

/* Return (chosen carrier). The carrier is replaced by the prepared observation
when cached_parse is active, so preparing cardinality never duplicates its
construction. Semantic dominance decisions are made elsewhere; cardinality-
dependent choices must pass through this function and retain their guard. */
(define ordered_scalar_recset_consumer_plan (lambda (src scalar_plan carrier offset limit)
	(begin
		(define source_rows (planner_source_row_count src))
		(define window_rows (+ (coalesceNil offset 0) (coalesceNil limit 0)))
		(define probe (physical_scalar_truth_plan_probe scalar_plan))
		(define stage (if (nil? probe) nil (car probe)))
		(define decision_id (concat "ordered_recset_consumer:"
			(if (nil? stage) (source_alias src) (gs_id stage))))
		/* Carrier producers are closed before reaching this consumer: direct boolean
		stages are RecSet algebra and ordinary membership carriers are relational
		plans. cached_parse can therefore observe the exact request-local cardinality
		without changing an outer-row binding or constructing the carrier twice. */
		(define observation_keys (planner_register_queryplan_observation decision_id
			(ordered_recset_observation_expr carrier)
			(list (quote recset_count) (symbol "__queryplan_observed_value"))))
		(define observed_rows (if (nil? observation_keys) nil
			(planner_queryplan_observed_metric decision_id)))
		(define planning_rows (if (number? observed_rows) observed_rows source_rows))
		(define direct_cost (ordered_direct_recset_cost planning_rows))
		(define base_cost (ordered_base_membership_cost source_rows planning_rows window_rows))
		(define normal_choice (if (planner_cost_better? direct_cost base_cost)
			"ordered_direct_recset" "ordered_base_membership"))
		(define crossover (if (number? source_rows)
			(ordered_recset_crossover_search source_rows window_rows 0 source_rows 32)
			nil))
		(if (or (nil? observation_keys) (not (number? crossover)))
			nil
			(planner_record_guard_condition
				(list (if (equal? normal_choice "ordered_direct_recset") (quote <) (quote >=))
					(list (quote session) (cadr observation_keys)) crossover)))
		(define chosen (planner_physical_choice decision_id normal_choice
			(list "ordered_direct_recset" "ordered_base_membership")))
		(define forced (planner_physical_override decision_id))
		(planner_record_physical_decision (list
			(list "decision_id" decision_id)
			(list "decision" "ordered_recset_consumer")
			(list "decision_site" "ordered_scan_lowering")
			(list "chosen" chosen)
			(list "selection" (if (nil? forced) "cost" "forced"))
			(list "reason" (if (nil? forced) "lowest_total_ns" "calibration_override"))
			(list "inputs" (list
				(list "carrier_rows" planning_rows)
				(list "driver_input_rows" source_rows)
				(list "offset" offset)
				(list "limit" limit)
				(list "crossover_rows" crossover)))
			(list "alternatives" (list
				(list
					(list "plan" "ordered_direct_recset")
					(list "status" (if (equal? chosen "ordered_direct_recset") "chosen" "rejected"))
					(list "cost" (planner_cost_explain direct_cost)))
				(list
					(list "plan" "ordered_base_membership")
					(list "status" (if (equal? chosen "ordered_base_membership") "chosen" "rejected"))
					(list "cost" (planner_cost_explain base_cost)))))))
		(list chosen (if (nil? observation_keys) carrier
			(list (quote session) (car observation_keys)))))))

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
				(define raw_condition (combine_where (qb_where block) (source_join_expr src)))
				(define order_items (coalesceNil (qb_order block) '()))
				(define scan_order_supported (order_items_belong_to_source? src order_items))
				(define bounded (query_limit_active? (qb_offset block) (qb_limit block)))
				/* The scalar carrier and the eventual scan must cost the same consumer.
				A native ordered LIMIT asks only for its bounded result window; an
				unorderable LIMIT cannot brake the source and retains full-table work. */
				(define probe_work_rows (if (and scan_order_supported bounded)
					(coalesceNil (probe_limit_work_rows (qb_limit block))
						(probe_context_row_count (list src)))
					(probe_context_row_count (list src))))
				(define scalar_plan (physical_scalar_truth_plan
					(list src) src alias raw_condition
					probe_work_rows (probe_context_row_count (list src))
					(query_block_stage_catalog block)))
				(define scalar_carrier (physical_scalar_truth_plan_carrier scalar_plan))
				(define scalar_consumer_plan (if (and (not (nil? scalar_carrier))
					(and scan_order_supported bounded))
					(ordered_scalar_recset_consumer_plan src scalar_plan scalar_carrier
						(qb_offset block) (qb_limit block))
					nil))
				(define scalar_consumer (if (nil? scalar_consumer_plan) nil
					(car scalar_consumer_plan)))
				(define effective_scalar_carrier (if (nil? scalar_consumer_plan)
					scalar_carrier (cadr scalar_consumer_plan)))
				(define scalar_membership_filter
					(equal? scalar_consumer "ordered_base_membership"))
				(define scalar_membership_var (symbol (concat "__ordered_scalar_recset_"
					(if (nil? (physical_scalar_truth_plan_probe scalar_plan))
						(source_alias src)
						(gs_id (car (physical_scalar_truth_plan_probe scalar_plan)))))))
				(define condition (rewrite_physical_scalar_truth_plan scalar_plan raw_condition))
				(define effective_fields (rewrite_physical_scalar_truth_plan scalar_plan fields))
				(define projection_bundle (physical_scalar_projection_bundle
					(list src) alias effective_fields))
				(define bundled_fields (if (nil? projection_bundle)
					effective_fields
					(nth projection_bundle 1)))
				(define source_table (source_table_expr_using (query_block_stage_catalog block) src))
				(define memberships (driver_memberships_for_source src condition))
				/* A derived Top-K has already moved ORDER/LIMIT into its row-number
				consumer. The logical membership requirement retains that ownership even
				when this block no longer carries a direct ORDER/LIMIT pair. */
				(define row_number_membership_consumer
					(membership_row_number_consumer? (if (empty_list? memberships)
						nil (car memberships)) (and scan_order_supported bounded)))
				(define allow_ordered_batch_binding (and scan_order_supported
					(and bounded
						(or (empty_list? order_items)
							(not (empty_list? (source_primary_key_columns src)))))))
				/* This node-local lowering is the sole carrier decision. Older code
				selected a keyset from block-level broadness facts before reaching this
				point; that duplicate choice could label an alternative as selected while
				emitting the correlated subscan fallback. */
				(define membership_plans (filter (map memberships (lambda (membership)
					(begin
						(define plan (recset_project_join_plan_for_membership_using
							src membership
							(if bounded (quote order_limit) (quote filter))
							(if (and scan_order_supported bounded)
								(probe_limit_work_rows (qb_limit block)) nil)
							allow_ordered_batch_binding
							(prefiltered_driver_recset_expr_for_membership
								src source_table condition membership)
							(+ (count (acceptance_required_sources
								(membership_downstream_sources (qb_sources block) src membership)
								alias raw_condition))
								(count (expr_probe_stages raw_condition)))
							(not row_number_membership_consumer)))
						(if (nil? plan) nil (list membership plan)))))
					(lambda (entry) (not (nil? entry)))))
				(define driver_memberships (map (filter membership_plans (lambda (entry)
					(equal? (car (nth entry 1)) "driver_order_membership_probe")))
					(lambda (entry) (nth entry 0))))
				/* For a bounded ordered driver, union branch-local candidates into one
				immutable key index. The chosen physical plans, not a logical fact or
				expression-shape heuristic, are the authority for this binding. */
				(define membership_keysets (membership_keyset_bindings driver_memberships))
				(define use_membership_keysets (and (not (empty_list? driver_memberships))
					(equal? (count membership_keysets) (count driver_memberships))))
				(define membership_bindings (filter (map membership_plans (lambda (entry)
					(begin
						(define membership (nth entry 0))
						(define plan (nth entry 1))
						(if (or (equal? (car plan) "candidate_keyset")
							(equal? (car plan) "prefiltered_candidate_keyset"))
							(list membership (membership_recset_var src membership) (cadr plan))
							nil))))
					(lambda (binding) (not (nil? binding)))))
				(define bound_memberships (map membership_bindings (lambda (binding) (nth binding 0))))
				(define membership_formula
					(driver_membership_recset_formula src condition membership_bindings))
				/* A membership predicate which is implied by the whole WHERE clause is
				eligible to become the scan driver. A branch-local predicate below OR is
				not: its RecSet must remain a probe or rows accepted by sibling branches
				would disappear. This distinction also preserves the established fast
				single-IN path while allowing several guarded RecSets in one scan. */
				(define direct_membership (driver_membership_for_source src condition))
				(define membership_formula_driver (not (nil? membership_formula)))
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
						(and (not (empty_list? membership_bindings))
							(and (empty_list? (cdr membership_bindings))
								(equal? direct_membership (car bound_memberships)))))))
				(define membership_filter (and
					(not (or membership_driver membership_formula_driver))
					(not (empty_list? membership_bindings))))
				(define candidate_filter_condition (if membership_formula_driver
					(if membership_formula_difference_driver true membership_formula_residual)
					(if membership_driver
						(strip_driver_membership_for_source src condition direct_membership)
						(replace_driver_membership_markers src condition bound_memberships))))
				(define filter_condition (if use_membership_keysets
					(replace_driver_membership_keyset_markers
						candidate_filter_condition membership_keysets)
					candidate_filter_condition))
				(define filtercols (merge_unique (list
					(if (or membership_filter scalar_membership_filter)
						(list "$recset_contains") '())
					(extract_columns_for_alias src filter_condition))))
				(define fieldcols (merge_unique (extract_assoc bundled_fields (lambda (_title expr)
					(extract_columns_for_alias src expr)))))
				(define ordercols (if (empty_list? order_items) '() (scan_order_sort_columns_for_alias src order_items)))
				(define batch_tiebreaker_cols (if (empty_list? order_items)
					'()
					(filter (source_primary_key_columns src)
						(lambda (col) (not (contains? ordercols col))))))
				(define batch_ordercols (merge (list ordercols batch_tiebreaker_cols)))
				(define batch_orderdirs (merge (list
					(if (empty_list? order_items) '() (order_relations_for_source src order_items))
					(map batch_tiebreaker_cols (lambda (col)
						(canonical_order_relation < (source_column_order_collation src col)))))))
				(define mapcols fieldcols)
				(define membership_candidates (if membership_filter
					(membership_or_candidate_recset src source_table condition membership_bindings)
					nil))
				(define membership_table_expr (if membership_formula_driver
					membership_formula_expr
					(if membership_driver
						(nth (car membership_bindings) 2)
						(coalesceNil membership_candidates source_table))))
				/* Independent scalar and membership edges may both select RecSet scan
				sources. Their conjunction is the exact physical intersection; replacing
				one with the other after either predicate was stripped would lose a WHERE
				term. */
				(define table_expr (if (or (nil? effective_scalar_carrier)
					scalar_membership_filter)
					membership_table_expr
					(if (equal? membership_table_expr source_table)
						effective_scalar_carrier
						(list (quote recset_intersect)
							(cons (quote list) (list
								effective_scalar_carrier
								membership_table_expr))))))
				(define filter_expr (list (quote lambda)
					(map filtercols (lambda (col) (scan_callback_symbol_for_alias alias col)))
					(if scalar_membership_filter
						(list (quote and)
							(recset_contains_call_expr scalar_membership_var)
							(lower_column_expr_for_alias src filter_condition))
						(lower_column_expr_for_alias src filter_condition))))
				(define remaining_batch_memberships (filter
					(ordered_batch_membership_terms src filter_condition)
					(lambda (membership) (ordered_batch_stage_supported? (nth membership 0)))))
				/* Membership markers which cannot project a batch RecSet remain in the
				residual expression and lower through the ordinary probe machinery. This
				makes adaptive ordered batching a property of the consuming scan, not of
				one privileged predicate shape. */
				(define use_batch_accept (and (equal? table_expr source_table)
					(reduce membership_plans (lambda (chosen entry)
						(or chosen (equal? (car (nth entry 1)) "ordered_batch_accept"))) false)))
				(define batch_filter (if use_batch_accept
					(ordered_batch_filter_expr (list src) alias src filter_condition
						remaining_batch_memberships (probe_context_row_count (list src)) '() true)
					nil))
				(define raw_map_row (list (quote resultrow)
					(cons (quote list) (map_assoc bundled_fields (lambda (title expr)
						(lower_column_expr_for_alias src expr))))))
				(define map_row (if (nil? projection_bundle)
					raw_map_row
					(list
						(list (quote lambda) (list (nth projection_bundle 0)) raw_map_row)
						(nth projection_bundle 2))))
				(define map_expr (list (quote lambda)
					(map mapcols (lambda (col) (symbol (concat alias "." col))))
					map_row))
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
							(define scan_expr (if use_batch_accept
								(list (quote scan_order_batch_accept)
									'(session "__memcp_tx")
									table_expr
									batch_filter
									(cons (quote list) batch_ordercols)
									(cons (quote list) batch_orderdirs)
									0
									(coalesceNil (qb_offset block) 0)
									(coalesceNil (qb_limit block) -1)
									(cons (quote list) mapcols)
									map_expr
									nil
									nil
									(source_outer? src))
								(list (quote scan_order)
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
									(source_outer? src))))
							scan_expr)
						(neumann_fail "build_queryplan" "single-source ORDER BY requires a storage carrier"))))
				(define wrapped_scan_plan (if use_membership_keysets
					(wrap_membership_keyset_bindings membership_keysets scan_plan)
					(if membership_filter
						(wrap_membership_recset_bindings membership_bindings scan_plan)
						scan_plan)))
				(if scalar_membership_filter
					(list
						(list (quote lambda) (list scalar_membership_var) wrapped_scan_plan)
						effective_scalar_carrier)
					wrapped_scan_plan))))))

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

(define scalar_projection_probe_entries_acc (lambda (expr state)
	(match expr
		((symbol scalar_first_probe) stage requested_col)
		(if (source_is_base_table? (gs_input stage))
			(begin
				(define key (concat (gs_id stage) "\n" requested_col))
				(if (has_assoc? (nth state 1) key)
					state
					(list
						(append (nth state 0) (list stage requested_col))
						(set_assoc (nth state 1) key true))))
			state)
		((quote scalar_first_probe) stage requested_col)
		(scalar_projection_probe_entries_acc
			(list (quote scalar_first_probe) stage requested_col) state)
		((symbol scalar_first_probe) stage requested_col _dependencies)
		(scalar_projection_probe_entries_acc
			(list (quote scalar_first_probe) stage requested_col) state)
		((quote scalar_first_probe) stage requested_col _dependencies)
		(scalar_projection_probe_entries_acc
			(list (quote scalar_first_probe) stage requested_col) state)
		(cons _head tail) (reduce tail (lambda (acc item)
			(scalar_projection_probe_entries_acc item acc)) state)
		_ state)))

(define scalar_projection_probe_entries (lambda (fields)
	(nth (scalar_projection_probe_entries_acc fields (list '() '())) 0)))

(define scalar_projection_bundle_index (lambda (entries stage requested_col)
	(reduce (produceN (count entries)) (lambda (found i)
		(if (not (nil? found)) found
			(begin
				(define entry (nth entries i))
				(if (and (equal? (gs_id (car entry)) (gs_id stage))
					(equal? (cadr entry) requested_col)) i nil)))) nil)))

(define rewrite_scalar_projection_bundle_expr (lambda (entries bundle_symbol expr)
	(match expr
		((symbol scalar_first_probe) stage requested_col) (begin
			(define i (scalar_projection_bundle_index entries stage requested_col))
			(if (nil? i) expr
				(list (quote if) (list (quote nil?) bundle_symbol) nil
					(list (quote nth) bundle_symbol i))))
		((quote scalar_first_probe) stage requested_col)
		(rewrite_scalar_projection_bundle_expr entries bundle_symbol
			(list (quote scalar_first_probe) stage requested_col))
		((symbol scalar_first_probe) stage requested_col _dependencies)
		(rewrite_scalar_projection_bundle_expr entries bundle_symbol
			(list (quote scalar_first_probe) stage requested_col))
		((quote scalar_first_probe) stage requested_col _dependencies)
		(rewrite_scalar_projection_bundle_expr entries bundle_symbol
			(list (quote scalar_first_probe) stage requested_col))
		(cons head tail) (cons head (map tail (lambda (item)
			(rewrite_scalar_projection_bundle_expr entries bundle_symbol item))))
		_ expr)))

(define physical_scalar_projection_bundle (lambda (sources default_alias fields)
	(begin
		(define entries (scalar_projection_probe_entries fields))
		(if (< (count entries) 2)
			nil
			(begin
				(define stage (car (car entries)))
				(define same_stage (reduce entries (lambda (same entry)
					(and same (equal? (gs_id (car entry)) (gs_id stage)))) true))
				(define ags (map entries (lambda (entry)
					(scalar_first_probe_aggregate (car entry) (cadr entry)))))
				(define parts (if (or (not same_stage)
					(reduce ags (lambda (missing ag) (or missing (nil? ag))) false))
					nil
					(map ags scalar_first_probe_parts)))
				(if (or (nil? parts)
					(not (compatible_scalar_order_aggregates? ags)))
					nil
					(begin
						(define src (gs_input stage))
						(define keys (gs_keys stage))
						(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
						(define condition (coalesceNil
							(qassoc_get (gs_facts stage) (quote condition) true) true))
						(define order_exprs (nth (car parts) 1))
						(define dirs (nth (car parts) 2))
						(define order_offset (nth (car parts) 3))
						(define value_exprs (map parts car))
						(define key_cols (merge_unique (map keys (lambda (expr)
							(extract_columns_for_alias src expr)))))
						(define order_cols (merge_unique (map order_exprs (lambda (expr)
							(extract_columns_for_alias src expr)))))
						(define filtercols (merge_unique (list
							(extract_columns_for_alias src condition)
							key_cols
							order_cols)))
						(define mapcols (merge_unique (map value_exprs (lambda (expr)
							(extract_columns_for_alias src expr)))))
						(define bundle_symbol (symbol "__scalar_projection_bundle"))
						(list
							bundle_symbol
							(rewrite_scalar_projection_bundle_expr entries bundle_symbol fields)
							(list (quote scan_order)
								'(session "__memcp_tx")
								(source_table_expr src)
								(cons (quote list) filtercols)
								(list (quote lambda)
									(map filtercols (lambda (col) (symbol (concat (source_alias src) "." col))))
									(cons (quote and) (cons
										(lower_column_expr_for_alias src condition)
										(scalar_first_probe_key_terms sources default_alias src keys lookup_keys))))
								(cons (quote list) order_cols)
								(cons (quote list) dirs)
								0 order_offset 1
								(cons (quote list) mapcols)
								(list (quote lambda)
									(map mapcols (lambda (col) (symbol (concat (source_alias src) "." col))))
									(cons (quote list) (map value_exprs (lambda (expr)
										(lower_column_expr_for_alias src expr)))))
								(scalar_once_reduce_first)
								nil false))))))))))

(define recset_contains_callback_symbol (symbol "__recset_contains"))

(define scan_callback_symbol_for_alias (lambda (alias col)
	(if (equal? col "$tx")
		(symbol col)
		(if (equal? col "$recset_contains")
			recset_contains_callback_symbol
			(symbol (concat alias "." col))))))

(define recset_contains_call_expr (lambda (recset_expr)
	(list recset_contains_callback_symbol recset_expr)))

/* A carrier selected for one conjunct changes the work seen by every later
conjunct. Exact query-local observations are the strongest fact; otherwise the
ordinary probe context remains the conservative input. This keeps AND
short-circuit ordering in the physical tree instead of encoding SQL shapes. */
(define membership_plan_residual_work_rows (lambda (src membership plan fallback)
	(if (nil? plan)
		fallback
		(begin
			(define strategy (car plan))
			(define decision_id (concat "membership_carrier:" (gs_id (nth membership 0))))
			(define observed_rows (planner_queryplan_observed_metric decision_id))
			(if (and (number? observed_rows)
				(or (equal? strategy "candidate_keyset")
					(equal? strategy "prefiltered_candidate_keyset")))
				observed_rows
				(if (equal? strategy "ordered_batch_accept")
					(batch_membership_survivor_rows (list membership) fallback)
					fallback))))))

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

/* Carrier costing happens at one scan-tree edge, but rejecting LEFT JOIN
sources may already sit in its continuation. Derive them from the complete
source catalog; exclude only the driver and the membership stage whose carrier
is being compared. acceptance_required_sources retains SQL NULL-extension
semantics and drops projection-only nullable lookups. */
(define membership_downstream_sources (lambda (sources driver_src membership)
	(filter (coalesceNil sources '()) (lambda (candidate)
		(and (not (equal? (source_alias candidate) (source_alias driver_src)))
			(not (and (stage_output_relation? (source_relation candidate))
				(equal? (stage_output_relation_id (source_relation candidate))
					(gs_id (nth membership 0))))))))))

/* The ordered root remains the storage driver while its continuation may emit
zero, one, or many complete values. stream_window_reduce owns SQL OFFSET/LIMIT
over those values, so neither driver cardinality nor a scalar-purpose tag can
move the window to the wrong tree level. */
(define join_ordered_streaming_limit_plan (lambda (schema all_sources plan default_alias needed_exprs final_condition order_items offset_value limit_value stages facts value_builder reduce_expr neutral_expr)
	(begin
		(define ordered_aliases (join_optimizer_tree_aliases plan))
		(define ordered_sources (join_optimizer_sources_for_order all_sources ordered_aliases))
		(define src (car ordered_sources))
		(define remaining_sources (cdr ordered_sources))
		(define alias (source_alias src))
		(define offset (coalesceNil offset_value 0))
		(define limit (coalesceNil limit_value -1))
		(define target (if (< limit 0) -1 (+ offset limit)))
		(define acceptance_probe_work_rows (coalesceNil
			(probe_limit_work_rows limit_value)
			(if (< target 0) nil target)))
		/* The ordered driver is consumed here. Preserve the optimizer's remaining
		subtree instead of rebuilding a left-deep tree from the source catalog. */
		(define remaining_plan (join_optimizer_tree_without_aliases
			plan (list (source_alias src))))
		(define raw_condition_parts (physical_partition_condition
			default_alias src remaining_sources final_condition))
		(define raw_condition (combine_where
			(source_join_expr src) (nth raw_condition_parts 0)))
		(define order_parts (split_order_items_for_join_driver
			ordered_sources default_alias src order_items stages final_condition '()))
		(define driver_order_items (nth order_parts 0))
		(define remaining_order_items (nth order_parts 1))
		(define membership (driver_membership_for_source src raw_condition))
		/* A RecSet batch represents driver membership, not join multiplicity. The
		consumer is therefore available exactly when the remaining join tree is a
		chain of 0:1/1:1 lookups. Predicate complexity is deliberately not a gate:
		the ordinary expression and probe lowerers run inside the smaller batch. */
		(define allow_ordered_batch (and (not (nil? membership))
			(and (>= target 0)
				(and (not (empty_list? driver_order_items))
					(and (ordered_batch_stage_supported? (nth membership 0))
						(and (not (expr_refs_stage_output_alias? final_condition))
							(downstream_sources_at_most_one_driver_row?
								ordered_sources default_alias final_condition stages)))))))
		(define membership_plan (if (nil? membership) nil
			(recset_project_join_plan_for_membership_using src membership
				(if (query_limit_active? offset_value limit_value) (quote order_limit) (quote filter))
				(if (query_limit_active? offset_value limit_value)
					(probe_limit_work_rows limit_value) nil)
				allow_ordered_batch
				(prefiltered_driver_recset_expr_for_membership
					src (source_table_expr_using stages src) raw_condition membership)
				(+ (count (acceptance_required_sources
					remaining_sources default_alias final_condition))
					(count (expr_probe_stages final_condition)))
				true)))
		(define membership_strategy (if (nil? membership_plan) nil (car membership_plan)))
		(define use_batch_accept (equal? membership_strategy "ordered_batch_accept"))
		/* A scalar truth carrier over the complete driver would defeat adaptive
		batching. When the batch alternative wins, keep the scalar marker in the
		predicate and lower it with the batch cardinality inside scan_recset. */
		(define scalar_plan (if use_batch_accept nil
			(physical_scalar_truth_plan
				all_sources src default_alias final_condition acceptance_probe_work_rows
				(planner_source_row_count src) stages)))
		(define scalar_carrier (physical_scalar_truth_plan_carrier scalar_plan))
		(define scalar_probe (physical_scalar_truth_plan_probe scalar_plan))
		(define carrier_condition (rewrite_physical_scalar_truth_plan scalar_plan final_condition))
		(define carrier_needed_exprs (rewrite_physical_scalar_truth_plan scalar_plan needed_exprs))
		(define condition_parts (physical_partition_condition
			default_alias src remaining_sources carrier_condition))
		(define local_condition (nth condition_parts 0))
		(define remaining_condition (combine_where
			(nth condition_parts 2)
			(join_optimizer_node_condition (join_optimizer_tree_predicates plan))))
		(define condition (combine_where (source_join_expr src) local_condition))
		(define membership_table_expr (if (and
			(not (nil? membership_plan))
			(or (equal? membership_strategy "candidate_keyset")
				(equal? membership_strategy "prefiltered_candidate_keyset")))
			(cadr membership_plan)
			nil))
		/* An ordered driver probe owns one immutable RHS key index. Reuse the
		same carrier representation as the single-source lowerer so join reordering
		cannot turn this alternative into one relational subscan per driver row. */
		(define membership_keysets (if (and (not (nil? membership))
			(equal? membership_strategy "driver_order_membership_probe"))
			(membership_keyset_bindings (list membership))
			'()))
		(define use_membership_keyset (not (empty_list? membership_keysets)))
		(define effective_membership (if (or use_batch_accept
			(not (nil? membership_table_expr))) membership nil))
		(define effective_condition (if use_membership_keyset
			(replace_driver_membership_keyset_markers condition membership_keysets)
			(strip_driver_membership_for_source src condition effective_membership)))
		/* The node-local physical choice is final. A full or driver-prefiltered
		candidate keyset becomes the ordered carrier. A driver-probe choice keeps the
		base table ordered and replaces a supported direct-column marker with the
		query-scoped key index; unsupported computed keys retain the established probe
		fallback and its separately calibrated cost. */
		(define filter_condition effective_condition)
		/* Stage-output column references require a relational source and cannot be
		read directly from the driver callback. Scalar/EXISTS probe markers are not
		deferred: lowering them at batch cardinality is the purpose of this path. */
		(define defer_complex_acceptance
			(expr_refs_stage_output_alias? remaining_condition))
		/* Acceptance runs before OFFSET/LIMIT and therefore contains only joins
		which can reject a driver row. Projection-only nullable lookups belong to
		the map callback after the native window and must not be probed twice. */
		(define acceptance_sources (if defer_complex_acceptance '()
			(acceptance_required_sources remaining_sources default_alias remaining_condition)))
		(define acceptance_aliases (map acceptance_sources source_alias))
		(define removed_acceptance_aliases (filter (map remaining_sources source_alias)
			(lambda (candidate) (not (contains? acceptance_aliases candidate)))))
		(define acceptance_plan (join_optimizer_tree_without_aliases
			remaining_plan removed_acceptance_aliases))
		(define acceptance_needed_exprs (merge (list
			(list remaining_condition)
			(source_join_exprs acceptance_sources))))
		(define acceptance_probe (if defer_complex_acceptance true
			(if (empty_list? acceptance_sources)
				(lower_column_expr_for_join_truth_context
					all_sources default_alias remaining_condition acceptance_probe_work_rows)
				(build_join_scan_reduce_using_recipe
					schema all_sources acceptance_plan default_alias acceptance_needed_exprs remaining_condition true
					/* This is EXISTS over complete rows, not LIMIT 1 on the first
					scan node. A driver candidate may have an early rejecting child and
					a later accepting child; only the outer ordered scan is allowed to
					brake after enough candidates have passed this complete reduction. */
					'() 0 -1 true acceptance_probe_work_rows nil stages
					(list (quote lambda) (list (quote _accepted) (quote _row)) true)
					false
					(list (quote lambda) (list (quote accepted) (quote shard_accepted))
						(list (quote or) (quote accepted) (quote shard_accepted)))
					nil))))
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
		(define batch_filter (if use_batch_accept
			(ordered_batch_filter_expr all_sources default_alias src condition
				(list membership) acceptance_probe_work_rows acceptance_cols acceptance_probe)
			nil))
		(if (and use_batch_accept (nil? batch_filter))
			(neumann_fail "build_queryplan" "chosen ordered batch membership has no executable filter")
			true)
		(define projection_probe_work_rows (coalesceNil
			(probe_limit_work_rows limit_value)
			acceptance_probe_work_rows))
		(define emit_value (quote __ordered_join_emit_value))
		(define row_expr (list (quote stream_emit) emit_value
			(value_builder projection_probe_work_rows scalar_probe)))
		(define mapcols (join_cols_for_alias all_sources default_alias alias carrier_needed_exprs))
		(define projection (build_join_scan_pipeline_using_recipe
			schema all_sources remaining_plan default_alias carrier_needed_exprs
			(if use_batch_accept true remaining_condition) row_expr
			remaining_order_items 0 -1 true projection_probe_work_rows nil stages nil))
		(define map_expr (list (quote lambda)
			(map mapcols (lambda (col) (scan_callback_symbol_for_alias alias col)))
			projection))
		(define ordercols (scan_order_sort_columns_for_join_driver
			ordered_sources default_alias src driver_order_items stages carrier_condition))
		(define tiebreaker_cols (if use_batch_accept
			(filter (source_primary_key_columns src)
				(lambda (col) (not (contains? ordercols col)))) '()))
		(define physical_ordercols (merge (list ordercols tiebreaker_cols)))
		(define physical_orderdirs (merge (list
			(order_relations_for_source src driver_order_items)
			(map tiebreaker_cols (lambda (col)
				(canonical_order_relation < (source_column_order_collation src col)))))))
		(define table_expr (if (nil? scalar_carrier)
			(coalesceNil membership_table_expr (source_table_expr_using stages src))
			scalar_carrier))
		(define scan_expr (if use_batch_accept
			(list (quote scan_order_batch_accept)
				'(session "__memcp_tx") table_expr batch_filter
				(cons (quote list) physical_ordercols)
				(cons (quote list) physical_orderdirs)
				0 offset limit
				(cons (quote list) mapcols) map_expr nil nil false)
			(list (quote scan_order)
				'(session "__memcp_tx") table_expr
				(cons (quote list) filtercols) filter_expr
				(cons (quote list) physical_ordercols)
				(cons (quote list) physical_orderdirs)
				0 0 (if defer_complex_acceptance -1 target)
				(cons (quote list) mapcols)
				map_expr nil nil false nil
				(cons (quote list) acceptance_cols)
				acceptance_expr)))
		(define window_expr (list (quote stream_window_reduce)
			(if use_batch_accept 0 offset) (if use_batch_accept -1 limit)
			reduce_expr neutral_expr
			(list (quote lambda) (list emit_value) scan_expr)))
		(if use_membership_keyset
			(wrap_membership_keyset_bindings membership_keysets window_expr)
			window_expr))))

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

/* Access-path enumerators describe executable alternatives; they do not pick
an operator, change join order, or remove a predicate. Adding a leaf-local
"if this expression then use that scan" below the common selector is a code
smell: add a descriptor here and let choose_scan_access_path compare it with
every other carrier instead. Logical join order remains owned by join_plan. */
(define scan_access_path_text_candidates (lambda (src all_sources default_alias condition)
	(begin
		(define alias (source_alias src))
		(define aliases (source_aliases all_sources))
		(define input_rows (planner_source_row_count src))
		(if (or (not (number? input_rows)) (< input_rows 1024))
			'()
			(filter
				(map (split_and_terms (coalesceNil condition true)) (lambda (term)
					(if (or (not (expr_contains_text_match? term))
						(not (equal? (join_hypergraph_expr_aliases default_alias aliases term)
							(list alias))))
						nil
						(begin
							(define estimate (planner_source_filter_estimate src term 512))
							(define rows (qassoc_get estimate (quote estimated_rows) nil))
							(define work (membership_source_work_profile src term true))
							(if (number? rows)
								(list
									(list (quote kind) (quote predicate_recset))
									(list (quote plan) "scan_recset")
									(list (quote predicate) term)
									(list (quote rows) rows)
									(list (quote input_rows) input_rows)
									(list (quote work) work)
									(list (quote estimate) estimate))
								nil)))))
				(lambda (item) (not (nil? item))))))))

(define scan_access_path_candidates (lambda (src all_sources default_alias condition)
	/* Text-backed exact RecSets are the first descriptor kind. Index ranges,
	persisted RecSets, or future scan primitives belong in this same list. */
	(scan_access_path_text_candidates src all_sources default_alias condition)))

/* The text predicate itself is common work. The carrier decision determines
whether the downstream join continuation is entered for every driver row or
only for exact matches after a parallel RecSet-producing pass. Reuse the
calibrated physical primitives; do not hide another selectivity threshold in
this lowering boundary. */
(define scan_access_path_scan_cost (lambda (src candidate)
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

(define scan_access_path_base_cost (lambda (src candidate)
	(begin
		(define input_rows (qassoc_get candidate (quote input_rows) 0))
		(planner_cost_add
			(scan_access_path_scan_cost src candidate)
			(planner_join_work_cost input_rows 0.65)
			(qassoc_get candidate (quote rows) input_rows) 0.65))))

(define scan_access_path_recset_cost (lambda (src candidate)
	(begin
		(define input_rows (qassoc_get candidate (quote input_rows) 0))
		(define rows (qassoc_get candidate (quote rows) input_rows))
		(planner_cost_add
			(planner_cost_add
				(scan_access_path_scan_cost src candidate)
				(planner_cost planner_membership_recset_startup_ns
					(* rows planner_membership_scan_row_ns) 0 0 0
					(* rows planner_membership_recset_build_row_ns)
					(* rows 8) 0 rows 0.65)
				rows 0.65)
			(planner_join_work_cost rows 0.65)
			rows 0.65))))

(define scan_access_path_candidate_cost (lambda (src candidate)
	(match (qassoc_get candidate (quote kind) nil)
		(quote predicate_recset) (scan_access_path_recset_cost src candidate)
		_ (neumann_fail "build_queryplan" "unsupported physical scan access-path descriptor"))))

/* Both alternatives contain the same text scan. Solve the remaining linear
cost inequality once and guard the cached plan by that crossover, not by an
exact parameter value. */
(define scan_access_path_crossover_rows (lambda (src candidate)
	(begin
		(define zero_candidate (qassoc_set candidate (quote rows) 0))
		(define one_candidate (qassoc_set candidate (quote rows) 1))
		(define recset_zero (qassoc_get
			(scan_access_path_recset_cost src zero_candidate) (quote total_ns) 0))
		(define recset_one (qassoc_get
			(scan_access_path_recset_cost src one_candidate) (quote total_ns) 0))
		(define base_total (qassoc_get
			(scan_access_path_base_cost src candidate) (quote total_ns) 0))
		(define row_slope (- recset_one recset_zero))
		(if (<= row_slope 0)
			0
			(max 0 (/ (- base_total recset_zero) row_slope))))))

(define scan_access_path_runtime_rows_expr (lambda (src candidate)
	(begin
		(define predicate (qassoc_get candidate (quote predicate) true))
		(define alias (source_alias src))
		(define cols (extract_columns_for_alias src predicate))
		(define pattern_expr (expr_text_pattern_expr predicate))
		(define estimate_expr (list (quote scan_selectivity_estimate)
			'(session "__memcp_tx")
			(list (quote table) (source_schema src) (source_relation src))
			(cons (quote list) cols)
			(list (quote lambda)
				(map cols (lambda (col) (scan_callback_symbol_for_alias alias col)))
				(lower_column_expr_for_alias src predicate))
			512))
		(list (quote planner_estimated_matching_rows)
			(if (nil? pattern_expr)
				estimate_expr
				(list (quote qassoc_set)
					estimate_expr
					(list (quote quote) (quote fallback_selectivity))
					(list (quote text_pattern_selectivity_prior) pattern_expr)))
			(qassoc_get candidate (quote input_rows) nil)
			(qassoc_get candidate (quote input_rows) nil)))))

(define choose_scan_access_path (lambda (src candidates)
	(if (empty_list? candidates)
		(list "fused_base_scan" nil)
		(begin
			(define candidate (reduce candidates (lambda (best item)
				(if (or (nil? best)
					(planner_cost_better?
						(scan_access_path_candidate_cost src item)
						(scan_access_path_candidate_cost src best)))
					item best)) nil))
			(define alias (source_alias src))
			(define decision_id (concat "scan_access_path:" alias))
			(define candidate_plan (qassoc_get candidate (quote plan) nil))
			(define candidate_cost (scan_access_path_candidate_cost src candidate))
			(define base_cost (scan_access_path_base_cost src candidate))
			(define work (qassoc_get candidate (quote work) '()))
			(define normal_choice (if (planner_cost_better? candidate_cost base_cost)
				candidate_plan "fused_base_scan"))
			(define crossover_rows (scan_access_path_crossover_rows src candidate))
			/* Guard the cost crossover even for a literal predicate. Auto-index
			construction can improve the same query's estimate after compilation;
			the cached variant must then be rejected if the winning path changes. */
			(planner_record_guard_condition
				(if (equal? normal_choice candidate_plan)
					(list (quote <)
						(scan_access_path_runtime_rows_expr src candidate)
						crossover_rows)
					(list (quote >=)
						(scan_access_path_runtime_rows_expr src candidate)
						crossover_rows)))
			(define alternatives (list candidate_plan "fused_base_scan"))
			(define chosen (planner_physical_choice decision_id normal_choice alternatives))
			(define forced (planner_physical_override decision_id))
			(planner_record_physical_decision (list
				(list "decision_id" decision_id)
				(list "decision" "scan_access_path")
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
						(qassoc_get candidate (quote input_rows) 1)))
					(list "crossover_rows" crossover_rows)))
				(list "alternatives" (list
					(list
						(list "plan" candidate_plan)
						(list "status" (if (equal? chosen candidate_plan) "chosen" "rejected"))
						(list "reason" (if (equal? chosen candidate_plan) "selected" "higher_total_ns_or_forced_alternative"))
						(list "cost" (planner_cost_explain candidate_cost)))
					(list
						(list "plan" "fused_base_scan")
						(list "status" (if (equal? chosen "fused_base_scan") "chosen" "rejected"))
						(list "reason" (if (equal? chosen "fused_base_scan") "selected" "higher_total_ns_or_forced_alternative"))
						(list "cost" (planner_cost_explain base_cost)))))))
			(list chosen candidate)))))

(define scan_access_path_recset_expr (lambda (stages src candidate)
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

(define scan_access_path_table_expr (lambda (stages src candidate)
	(match (qassoc_get candidate (quote kind) nil)
		(quote predicate_recset) (scan_access_path_recset_expr stages src candidate)
		_ (neumann_fail "build_queryplan" "unsupported physical scan access-path descriptor"))))

(define strip_scan_access_path_predicate (lambda (condition candidate)
	(if (nil? candidate)
		condition
		(begin
			(define predicate (qassoc_get candidate (quote predicate) true))
			(combine_where_terms
				(filter (split_and_terms (coalesceNil condition true)) (lambda (term)
					(not (equal? term predicate))))
				true)))))

(define build_join_scan_leaf_using_recipe (lambda (schema all_sources leaf future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_context scalar_plan continuation outer_scan)
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
			default_alias final_condition offset_value limit_value stages))
		(define allow_ordered_batch (and (not (nil? membership))
			(and (not delay_limit_after_join)
				(and (not (empty_list? current_order_items))
					(and (query_limit_active? offset_value limit_value)
						(and (ordered_batch_stage_supported? (nth membership 0))
							(downstream_sources_at_most_one_driver_row?
								ordered_sources default_alias final_condition stages)))))))
		/* A row-number pipeline owns the order/limit continuation itself. Its
		logical requirement survives ORC/window lowering, whereas the stage may no
		longer be present in this leaf's local catalog. It can consume a projected
		RecSet but cannot execute a separate ordered-driver membership probe without
		first materializing the derived window. */
		(define direct_order_limit (and (not (empty_list? current_order_items))
			(query_limit_active? offset_value limit_value)))
		(define row_number_membership_consumer
			(membership_row_number_consumer? membership direct_order_limit))
		(define membership_plan (if (or (nil? membership) (or delay_limit_after_join (not allow_membership_recset)))
			nil
			(recset_project_join_plan_for_membership_using src membership
				(if (join_scan_reduce? result_mode) (quote aggregate)
					(if (or direct_order_limit row_number_membership_consumer)
						(quote order_limit) (quote filter)))
				(if (and (not (empty_list? current_order_items))
					(query_limit_active? offset_value limit_value))
					(probe_limit_work_rows limit_value) nil)
				allow_ordered_batch
				(prefiltered_driver_recset_expr_for_membership
					src (source_table_expr_using stages src) condition membership)
				(+ (count (acceptance_required_sources
					(membership_downstream_sources all_sources src membership)
					default_alias final_condition))
					(count (expr_probe_stages final_condition)))
				(not row_number_membership_consumer))))
		(define membership_strategy (if (nil? membership_plan) nil (car membership_plan)))
		(define use_batch_accept (equal? membership_strategy "ordered_batch_accept"))
		(define residual_probe_work_rows (membership_plan_residual_work_rows
			src membership membership_plan
			(probe_work_context_rows_for_alias probe_context alias)))
		/* A preceding membership carrier may radically reduce this leaf. Choose
		the scalar/EXISTS carrier only now, with that node-local cardinality. An
		adaptive batch keeps the marker inside its batch predicate so each batch
		runs the same lowerer with its own work estimate. */
		(define effective_scalar_plan (if (not (nil? scalar_plan))
			scalar_plan
			/* Without a preceding membership carrier, retain the established
			leaf-condition lowering. It owns nullable LEFT-JOIN cardinality and must
			not be pre-empted by a driver-context estimate from another tree node. */
			(if (or (nil? membership_plan) use_batch_accept) nil
				(physical_scalar_truth_plan all_sources src default_alias condition
					residual_probe_work_rows residual_probe_work_rows stages))))
		(define effective_scalar_carrier
			(physical_scalar_truth_plan_carrier effective_scalar_plan))
		(define scalar_carrier_driver (and (not (nil? effective_scalar_carrier))
			(equal? alias (car effective_scalar_plan))))
		/* An exact truth RecSet does not imply that iterating that RecSet is the
		cheapest ordered access path. At the selected join-tree driver, compare the
		complete projected carrier with the base index plus one membership test per
		visited row. This is the same node-local choice used by a single-source
		block; keeping it here makes added projection/LEFT-JOIN leaves irrelevant to
		the operator decision. Cardinality-dependent choices retain the observation
		and recompile guard installed by ordered_scalar_recset_consumer_plan. */
		(define scalar_consumer_plan (if (and scalar_carrier_driver
			(and direct_order_limit (not delay_limit_after_join)))
			(ordered_scalar_recset_consumer_plan src effective_scalar_plan
				effective_scalar_carrier offset_value limit_value)
			nil))
		(define scalar_consumer (if (nil? scalar_consumer_plan) nil
			(car scalar_consumer_plan)))
		(define consumed_scalar_carrier (if (nil? scalar_consumer_plan)
			effective_scalar_carrier (cadr scalar_consumer_plan)))
		(define scalar_membership_filter
			(equal? scalar_consumer "ordered_base_membership"))
		(define scalar_membership_var (symbol (concat "__ordered_scalar_recset_"
			(if (nil? (physical_scalar_truth_plan_probe effective_scalar_plan))
				alias
				(gs_id (car (physical_scalar_truth_plan_probe effective_scalar_plan)))))))
		(define carrier_condition
			(rewrite_physical_scalar_truth_plan effective_scalar_plan condition))
		(define membership_table_expr (if (and
			(not (nil? membership_plan))
			(or (equal? membership_strategy "candidate_keyset")
				(equal? membership_strategy "prefiltered_candidate_keyset")))
			(cadr membership_plan)
			nil))
		/* Join-tree leaves consume the same ordered RHS key-index alternative as
		the top-level ordered lowerer. The index is bound outside the scan callback
		and is therefore built once per query, irrespective of tree depth. */
		(define membership_keysets (if (and (not (nil? membership))
			(equal? membership_strategy "driver_order_membership_probe"))
			(membership_keyset_bindings (list membership))
			'()))
		(define use_membership_keyset (not (empty_list? membership_keysets)))
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
					(list "keyset_binding_count" (count membership_keysets))
					(list "carrier_allowed" allow_membership_recset)
					(list "limit_delayed" delay_limit_after_join)
					(list "projection_built" (not (nil? membership_table_expr)))))))
			nil)
		(define effective_membership (if (or use_batch_accept
			(not (nil? membership_table_expr))) membership nil))
		(define effective_condition (if use_membership_keyset
			(replace_driver_membership_keyset_markers carrier_condition membership_keysets)
			(strip_driver_membership_for_source src carrier_condition effective_membership)))
		(define row_number_stage_filter (row_number_stage_for_source stages src effective_condition))
		/* A candidate-keyset choice makes the projected row positions this leaf's
		scan carrier even when later join leaves are continuations. A supported
		driver-probe choice instead keeps the base table and binds one RHS key index
		outside every continuation. The row-number pipeline still owns a base-table
		carrier, so it consumes a chosen candidate RecSet as a membership filter until
		it accepts an explicit source. */
		(define membership_driver (and
			(not (nil? membership_table_expr))
			(nil? row_number_stage_filter)))
		(define membership_filter (and (not (nil? membership_table_expr)) (not membership_driver)))
		/* Query-local access-path construction is safe only at the selected driver:
		inside a continuation it would rebuild the carrier once per outer row. This
		is an execution-scope capability, not a predicate- or text-specific rule.
		Join reorder has already selected the driver before this boundary. */
		(define access_path_build_allowed (and
			(not membership_driver)
			(nil? row_number_stage_filter)
			(> (count all_sources) 1)
			(equal? alias (probe_work_context_driver_alias probe_context))))
		(define access_path_candidates_before_point_check (if access_path_build_allowed
			(scan_access_path_candidates src all_sources default_alias effective_condition)
			'()))
		/* A unique point lookup already bounds downstream work. This capability
		check applies uniformly to every candidate; an enumerator must not hide a
		second operator decision behind its expression-shape recognizer. */
		(define access_path_candidates (if (and
			(not (empty_list? access_path_candidates_before_point_check))
			(source_unique_point_condition? src effective_condition))
			'()
			access_path_candidates_before_point_check))
		(define access_path_plan (choose_scan_access_path src access_path_candidates))
		(define access_path_candidate (cadr access_path_plan))
		(define access_path_selected (and (not (nil? access_path_candidate))
			(equal? (car access_path_plan)
				(qassoc_get access_path_candidate (quote plan) nil))))
		(define residual_condition
			(strip_scan_access_path_predicate effective_condition
				(if access_path_selected access_path_candidate nil)))
		(define membership_var (symbol "__membership_recset"))
		(define membership_filter_expr (if membership_filter
			(recset_contains_call_expr membership_var)
			true))
		(define scalar_membership_filter_expr (if scalar_membership_filter
			(recset_contains_call_expr scalar_membership_var)
			true))
		(define filter_condition (combine_where scalar_membership_filter_expr
			(combine_where membership_filter_expr residual_condition)))
		(define filtercols (merge_unique (list
			(if (or membership_filter scalar_membership_filter)
				(list "$recset_contains") '())
			(join_filter_cols_for_alias all_sources default_alias alias residual_condition))))
		(define recipe_mapcols (join_recipe_mapcols column_recipe alias))
		(define raw_mapcols (if (nil? column_recipe)
			(join_cols_for_alias all_sources default_alias alias needed_exprs)
			recipe_mapcols))
		(define mapcols raw_mapcols)
		(define base_table_expr (if membership_driver membership_table_expr
			(if (not access_path_selected)
				(source_table_expr_using stages src)
				(scan_access_path_table_expr stages src access_path_candidate))))
		(define table_expr (if (or (not scalar_carrier_driver) scalar_membership_filter)
			base_table_expr
			(if (equal? base_table_expr (source_table_expr_using stages src))
				consumed_scalar_carrier
				(list (quote recset_intersect)
					(cons (quote list) (list
						consumed_scalar_carrier
						base_table_expr))))))
		(define lowered_filter_condition (mark_outer_join_symbols
			all_sources
			alias
			(lower_column_expr_for_join_truth_context
				all_sources default_alias filter_condition
				(probe_work_context_rows_for_alias probe_context alias))))
		(define filter_expr (list (quote lambda)
			(map filtercols (lambda (col) (scan_callback_symbol_for_alias alias col)))
			lowered_filter_condition))
		(define batch_filter (if use_batch_accept
			(ordered_batch_filter_expr all_sources default_alias src condition
				(list membership) (probe_work_context_rows_for_alias probe_context alias) '() true)
			nil))
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
				(if use_batch_accept
					(begin
						(define ordercols (scan_order_sort_columns_for_join_driver
							ordered_sources default_alias src current_order_items stages final_condition))
						(define tiebreaker_cols (filter (source_primary_key_columns src)
							(lambda (col) (not (contains? ordercols col)))))
						(list (quote scan_order_batch_accept)
							'(session "__memcp_tx")
							table_expr batch_filter
							(cons (quote list) (merge (list ordercols tiebreaker_cols)))
							(cons (quote list) (merge (list
								(order_relations_for_source src current_order_items)
								(map tiebreaker_cols (lambda (col)
									(canonical_order_relation <
										(source_column_order_collation src col)))))))
							0 (coalesceNil offset_value 0) (coalesceNil limit_value -1)
							(cons (quote list) mapcols) map_expr reduce_expr
							(join_scan_neutral_expr result_mode)
							(or outer_scan (source_outer? src))))
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
							(or outer_scan (source_outer? src)))))))
		(define membership_bound_scan (if membership_filter
			(list
				(list (quote lambda) (list membership_var) scan_expr)
				membership_table_expr)
			scan_expr))
		(define scalar_bound_scan (if scalar_membership_filter
			(list
				(list (quote lambda) (list scalar_membership_var) membership_bound_scan)
				consumed_scalar_carrier)
			membership_bound_scan))
		(if use_membership_keyset
			(wrap_membership_keyset_bindings membership_keysets scalar_bound_scan)
			scalar_bound_scan))))

/* Consume the logical join tree recursively. The right subtree is lowered as
the continuation of the left subtree, so join-node boundaries and outer-join
ownership remain available until the physical scans are emitted. */
(define build_join_tree_scan_using_recipe (lambda (schema all_sources tree future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_context scalar_plan continuation outer_scan)
	(match tree
		((symbol join-leaf) _alias)
		(build_join_scan_leaf_using_recipe schema all_sources tree future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_context scalar_plan continuation outer_scan)
		((quote join-leaf) alias)
		(build_join_tree_scan_using_recipe schema all_sources (make_join_optimizer_leaf alias) future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_context scalar_plan continuation outer_scan)
		((symbol join-leaf) _alias _predicates)
		(build_join_scan_leaf_using_recipe schema all_sources tree future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_context scalar_plan continuation outer_scan)
		((quote join-leaf) alias predicates)
		(build_join_tree_scan_using_recipe schema all_sources
			(list (quote join-leaf) alias predicates) future_aliases default_alias needed_exprs
			final_condition row_expr order_items offset_value limit_value allow_membership_recset
			column_recipe stages result_mode probe_context scalar_plan continuation outer_scan)
		((symbol join-node) kind left right predicates)
		(begin
			/* NULL extension is a property of the LEFT boundary, independent of
			whether its nullable relation is one leaf or a complete reordered subtree.
			The recursive scan carries outer_scan into that subtree; its neutral path
			emits exactly one all-NULL composite row when no descendant row satisfies
			the boundary predicate. Nested LEFT nodes retain their own boundary and
			therefore perform their own NULL extension without materialization. */
			(define right_aliases (join_optimizer_tree_aliases right))
			(define node_condition (join_optimizer_node_condition predicates))
			(build_join_tree_scan_using_recipe
				schema all_sources left (merge_unique (list right_aliases future_aliases))
				/* Feed node predicates through the same partitioning walk as residual
				WHERE terms. Besides evaluating them at the first fully-bound node, this
				makes a rejecting downstream predicate visible before the left carrier
				chooses whether its native LIMIT is semantically safe. */
				default_alias needed_exprs (combine_where final_condition node_condition) row_expr
				order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_context scalar_plan
				(lambda (left_condition left_row_expr left_order_items)
					(build_join_tree_scan_using_recipe
						schema all_sources right future_aliases
						default_alias needed_exprs left_condition left_row_expr
						left_order_items 0 -1 allow_membership_recset column_recipe stages result_mode probe_context scalar_plan continuation
						(equal? kind (quote left-outer))))
				outer_scan))
		((quote join-node) kind left right predicates)
		(build_join_tree_scan_using_recipe schema all_sources
			(make_join_optimizer_node kind left right predicates) future_aliases
			default_alias needed_exprs final_condition row_expr order_items offset_value limit_value
			allow_membership_recset column_recipe stages result_mode probe_context scalar_plan continuation outer_scan)
		_ (neumann_fail "build_queryplan" "malformed logical join tree"))))

(define build_join_scan_with_mapper_using_recipe (lambda (schema all_sources sources default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_work_rows scalar_plan)
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
				scalar_plan terminal false)))))

(define build_join_scan_pipeline_using_recipe (lambda (schema all_sources sources default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_carrier probe_work_rows column_recipe stages scalar_plan)
	(build_join_scan_with_mapper_using_recipe
		schema all_sources sources default_alias needed_exprs final_condition row_expr
		order_items offset_value limit_value allow_membership_carrier column_recipe stages
		(list (quote pipeline)) probe_work_rows scalar_plan)))

(define build_join_scan_reduce_using_recipe (lambda (schema all_sources sources default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_carrier probe_work_rows column_recipe stages reduce_expr neutral_expr shard_reduce_expr scalar_plan)
	(build_join_scan_with_mapper_using_recipe
		schema all_sources sources default_alias needed_exprs final_condition row_expr
		order_items offset_value limit_value allow_membership_carrier column_recipe stages
		(list (quote reduce) reduce_expr neutral_expr shard_reduce_expr) probe_work_rows scalar_plan)))

(define build_join_scan_sink (lambda (schema sources default_alias needed_exprs final_condition sink_expr stages)
	(build_join_scan_pipeline_using_recipe
		schema sources sources default_alias needed_exprs final_condition sink_expr
		'() 0 -1 false nil nil stages nil)))

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
			(concat "s" pos "_" (stable_structural_hash col false))))))

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
		(concat ".prejoin:" (stable_structural_hash signature true)))))

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

/* A scalar truth carrier is selected only for a complete positive WHERE
conjunct. Finding it here keeps the logical join tree untouched and lets the
consumer compare direct, keytable, and projected-RecSet costs exactly once.
COALESCE(..., FALSE) is the common SQL-3VL truth-filter wrapper; choosing one
conjunct of AND is safe because every surviving row must satisfy it. */
(define physical_scalar_truth_probe_term (lambda (expr)
	(match expr
		((symbol scalar_first_probe) stage requested_col)
		(list stage requested_col (list stage))
		((quote scalar_first_probe) stage requested_col)
		(list stage requested_col (list stage))
		((symbol scalar_first_probe) stage requested_col dependencies)
		(list stage requested_col dependencies)
		((quote scalar_first_probe) stage requested_col dependencies)
		(list stage requested_col dependencies)
		((symbol coalesceNil) inner false)
		(physical_scalar_truth_probe_term inner)
		((quote coalesceNil) inner false)
		(physical_scalar_truth_probe_term inner)
		((symbol optimize) inner)
		(physical_scalar_truth_probe_term inner)
		((quote optimize) inner)
		(physical_scalar_truth_probe_term inner)
		_ nil)))

(define physical_scalar_truth_probe (lambda (condition)
	(reduce (split_and_terms (coalesceNil condition true)) (lambda (found term)
		(if (not (nil? found)) found
			(physical_scalar_truth_probe_term term))) nil)))

(define physical_scalar_probe_matches? (lambda (probe stage requested_col)
	(and (equal? (gs_id (car probe)) (gs_id stage))
		(equal? (cadr probe) requested_col))))

(define rewrite_physical_scalar_probe_as_true (lambda (probe expr)
	(match expr
		((symbol scalar_first_probe) stage requested_col)
		(if (physical_scalar_probe_matches? probe stage requested_col) true expr)
		((quote scalar_first_probe) stage requested_col)
		(if (physical_scalar_probe_matches? probe stage requested_col) true expr)
		((symbol scalar_first_probe) stage requested_col _dependencies)
		(if (physical_scalar_probe_matches? probe stage requested_col) true expr)
		((quote scalar_first_probe) stage requested_col _dependencies)
		(if (physical_scalar_probe_matches? probe stage requested_col) true expr)
		(cons head tail) (cons head (map tail (lambda (item)
			(rewrite_physical_scalar_probe_as_true probe item))))
		_ expr)))

(define scalar_probe_bound_lookup_keys (lambda (driver_src condition lookup_keys)
	(reverse (nth (reduce lookup_keys (lambda (state key)
		(if (not (car state))
			state
			(begin
				(define col (direct_column_name_for_alias driver_src key))
				(if (nil? col)
					(if (expr_contains_column_ref? key)
						(list false '())
						(list true (cons key (cadr state))))
					(begin
						(define binding (source_column_equality_binding driver_src col condition))
						(if (car binding)
							(list true (cons (cadr binding) (cadr state)))
							(list false '())))))))
		(list true '())) 1))))

(define scalar_probe_stage_with_bound_lookup_keys (lambda (driver_src condition stage)
	(begin
		(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
		(define bound_keys (scalar_probe_bound_lookup_keys driver_src condition lookup_keys))
		(if (or (empty_list? lookup_keys) (not (equal? (count bound_keys) (count lookup_keys))))
			nil
			(group_stage_with_facts stage
				(qassoc_set
					(qassoc_set (gs_facts stage) (quote lookup-keys) bound_keys)
					(quote segment_invariant_scalar_probe) true))))))

(define rewrite_physical_scalar_probe_stage (lambda (probe replacement expr)
	(match expr
		((symbol scalar_first_probe) stage requested_col)
		(if (physical_scalar_probe_matches? probe stage requested_col)
			(list (quote scalar_first_probe) replacement requested_col)
			expr)
		((quote scalar_first_probe) stage requested_col)
		(rewrite_physical_scalar_probe_stage probe replacement
			(list (symbol "scalar_first_probe") stage requested_col))
		((symbol scalar_first_probe) stage requested_col dependencies)
		(if (physical_scalar_probe_matches? probe stage requested_col)
			(list (quote scalar_first_probe) replacement requested_col
				(map dependencies (lambda (dependency)
					(if (and (group_stage? dependency) (equal? (gs_id dependency) (gs_id stage)))
						replacement dependency))))
			expr)
		((quote scalar_first_probe) stage requested_col dependencies)
		(rewrite_physical_scalar_probe_stage probe replacement
			(list (symbol "scalar_first_probe") stage requested_col dependencies))
		(cons head tail) (cons head (map tail (lambda (item)
			(rewrite_physical_scalar_probe_stage probe replacement item))))
		_ expr)))

(define physical_scalar_truth_plan_carrier (lambda (plan)
	(if (nil? plan) nil (nth plan 1))))

(define physical_scalar_truth_plan_probe (lambda (plan)
	(if (or (nil? plan) (nil? (physical_scalar_truth_plan_carrier plan))) nil (nth plan 2))))

(define rewrite_physical_scalar_truth_plan (lambda (plan expr)
	(if (nil? plan)
		expr
		(if (not (nil? (physical_scalar_truth_plan_carrier plan)))
			(rewrite_physical_scalar_probe_as_true (nth plan 2) expr)
			(rewrite_physical_scalar_probe_stage (nth plan 2) (nth plan 3) expr)))))

/* Return (driver-alias carrier-expression original-probe selected-stage).
The carrier is non-nil only for the projected RecSet alternative. When every
correlation key is fixed by a source-local equality, selected-stage instead
records the semantic proof that its value is constant for the whole base-index
segment even though the logical expression remains correlated.

That proof is an immediate dominance decision: evaluating and caching the one
scalar cannot require more driver work than projecting a full carrier. Do not
extend this branch with estimated-selectivity or row-count thresholds. Whenever
either alternative can win for different data, feed both costs into the normal
physical decision and preserve its runtime recompile gate. */
(define physical_scalar_truth_plan (lambda (sources driver_src default_alias condition probe_work_rows carrier_work_rows stages)
	(begin
		(define probe (physical_scalar_truth_probe condition))
		(if (nil? probe)
			nil
			(begin
				(define raw_stage (car probe))
				(define requested_col (cadr probe))
				(define dependencies (nth probe 2))
				(define bounded_direct_context (and
					(number? (planner_literal_value probe_work_rows))
					(and (number? (planner_literal_value carrier_work_rows))
						(< (planner_literal_value probe_work_rows)
							(planner_literal_value carrier_work_rows)))))
				(define bound_stage (if bounded_direct_context
					(scalar_probe_stage_with_bound_lookup_keys driver_src condition raw_stage)
					nil))
				(define stage (coalesceNil bound_stage raw_stage))
				(define src (gs_input stage))
				(define raw_lookup_keys (qassoc_get (gs_facts raw_stage) (quote lookup-keys) '()))
				(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
				(define keys (if (empty_list? lookup_keys) '() (gs_keys stage)))
				(define carrier_src (scalar_first_probe_carrier_source src))
				(define key_index (if (or (nil? carrier_src)
					(not (equal? (count keys) (count raw_lookup_keys))))
					nil
					(scalar_first_probe_keytable_key_index raw_stage carrier_src keys)))
				(define target_col (if (nil? key_index) nil
					(direct_column_name_for_alias driver_src (nth raw_lookup_keys key_index))))
				(define probe_stages (stage_catalog_with_nested
					(merge_stage_catalogs (list stages dependencies
						(qassoc_get (gs_facts raw_stage) (quote probe_catalog) '())
						(list raw_stage)))))
				/* LIMIT bounds returned rows, not necessarily rejected candidates. Only a
				segment-constant probe is guaranteed to execute once. A row-varying ACL may
				inspect the complete segment before filling the window, so retain the full
				carrier workload for that comparison. */
				(define effective_probe_work_rows (if (nil? bound_stage)
					(if bounded_direct_context carrier_work_rows probe_work_rows)
					1))
				(define operator (if (nil? target_col) (quote unsupported)
					(scalar_first_probe_physical_operator
						probe_stages
						(stage_dependency_graph probe_stages)
						raw_stage src keys effective_probe_work_rows carrier_work_rows requested_col (quote truth))))
				(if (and (not (equal? operator (quote recset))) (nil? bound_stage))
					nil
					(list
						(source_alias driver_src)
						(if (equal? operator (quote recset))
							(lower_projected_recset_scalar_first_probe_expr
								probe_stages raw_stage requested_col driver_src target_col true)
							nil)
						probe
						(if (equal? operator (quote recset)) raw_stage stage))))))))

(define build_join_scan_rows (lambda (schema sources plan default_alias needed_exprs final_condition fields order_items offset_value limit_value limit_brakes stages facts)
	(begin
		(define driver_source (join_optimizer_source_by_alias sources
			(join_optimizer_tree_first_alias plan)))
		(define filter_probe_work_rows (membership_probe_work_rows facts final_condition
			(planner_row_count_after_selectivity
				driver_source sources default_alias final_condition nil)))
		/* A native ordered LIMIT bounds every consumer inside the driver scan,
		including rejecting scalar probes. Cost that continuation by the rows the
		window requests, just as join_ordered_streaming_limit_plan does. Plans whose
		ORDER cannot brake retain the complete filtered workload. */
		(define bounded_probe_work_rows (if limit_brakes
			(coalesceNil (probe_limit_work_rows limit_value) filter_probe_work_rows)
			filter_probe_work_rows))
		/* Membership is an AND-carrier alternative, not merely a predicate leaf.
		When present at the driver, its physical choice must establish the rows
		seen by an expensive scalar/EXISTS conjunct before that conjunct chooses
		between probes and a complete RecSet. The leaf owns that second choice. */
		(define defer_scalar_carrier
			(not (nil? (driver_membership_for_source driver_source final_condition))))
		(define scalar_plan (if defer_scalar_carrier nil
			(physical_scalar_truth_plan
				sources driver_source default_alias final_condition
				bounded_probe_work_rows filter_probe_work_rows stages)))
		(define effective_condition (rewrite_physical_scalar_truth_plan scalar_plan final_condition))
		(define effective_fields (rewrite_physical_scalar_truth_plan scalar_plan fields))
		(define effective_needed_exprs (rewrite_physical_scalar_truth_plan scalar_plan needed_exprs))
		(define projection_probe_work_rows (if (query_limit_active? offset_value limit_value)
			(coalesceNil (probe_limit_work_rows limit_value) 0)
			(if (probe_context_unique_point? sources default_alias effective_condition)
				1
				filter_probe_work_rows)))
		(define projection_bundle (physical_scalar_projection_bundle
			sources default_alias effective_fields))
		(define bundled_fields (if (nil? projection_bundle)
			effective_fields
			(nth projection_bundle 1)))
		(define raw_row_expr (list (quote resultrow)
			(cons (quote list) (lower_join_result_fields
				sources default_alias bundled_fields projection_probe_work_rows))))
		(define row_expr (if (nil? projection_bundle)
			raw_row_expr
			(list
				(list (quote lambda) (list (nth projection_bundle 0)) raw_row_expr)
				(nth projection_bundle 2))))
		(build_join_scan_pipeline_using_recipe
			schema sources plan default_alias effective_needed_exprs effective_condition row_expr
			order_items offset_value limit_value true bounded_probe_work_rows nil stages scalar_plan))))

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
		(define ordered_sources (join_optimizer_sources_for_order scan_sources
			(join_optimizer_tree_aliases scan_plan)))
		(define direct_order_safe (and direct_order
			(not (ordered_join_limit_requires_complete_rows?
				ordered_sources first_alias final_condition (qb_offset block) (qb_limit block)
				(query_block_stage_catalog block)))))
		(define hierarchical_order (order_items_follow_join_tree?
			ordered_sources first_alias order_items (query_block_stage_catalog block) final_condition))
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
		(define scalar_carrier_probe_work_rows (if direct_order_safe
			projection_probe_work_rows
			unbounded_probe_work_rows))
		(if (or direct_order_safe
			(and hierarchical_order
				(driver_limit_cannot_brake?
					ordered_sources first_alias final_condition
					(qb_offset block) (qb_limit block) (query_block_stage_catalog block))))
			(begin
				(define scalar_plan (physical_scalar_truth_plan
					scan_sources driver_source first_alias final_condition
					scalar_carrier_probe_work_rows unbounded_probe_work_rows
					(query_block_stage_catalog block)))
				(define effective_condition (rewrite_physical_scalar_truth_plan scalar_plan final_condition))
				(define effective_field_exprs (rewrite_physical_scalar_truth_plan scalar_plan field_exprs))
				(define effective_needed_exprs (rewrite_physical_scalar_truth_plan scalar_plan needed_exprs))
				(define row_expr (cons row_mapper (map effective_field_exprs (lambda (expr)
					(lower_column_expr_for_join_in_context
						scan_sources first_alias expr projection_probe_work_rows)))))
				(build_join_scan_reduce_using_recipe
					(qb_schema block)
					scan_sources
					scan_plan
					first_alias
					effective_needed_exprs
					effective_condition
					row_expr
					order_items
					(coalesceNil (qb_offset block) 0)
					(coalesceNil (qb_limit block) -1)
					true projection_probe_work_rows nil (query_block_stage_catalog block)
					reduce_expr neutral_expr shard_reduce_expr scalar_plan))
			(if hierarchical_order
				(join_ordered_streaming_limit_plan
					(qb_schema block) scan_sources scan_plan first_alias needed_exprs final_condition
					order_items (qb_offset block) (qb_limit block) (query_block_stage_catalog block)
					(qb_facts block)
					(lambda (probe_work_rows scalar_probe)
						(cons row_mapper (map (if (nil? scalar_probe) field_exprs
							(rewrite_physical_scalar_probe_as_true scalar_probe field_exprs)) (lambda (expr)
								(lower_column_expr_for_join_in_context
									scan_sources first_alias expr probe_work_rows)))))
					reduce_expr neutral_expr)
				(neumann_fail "build_queryplan"
					"ordered dataset reduction has no streamable join-tree order"))))))

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
			(not (scalar_value_stage? stage))))
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
					(not (ordered_join_limit_requires_complete_rows? ordered_sources first_alias final_condition
						(qb_offset block) (qb_limit block) stage_catalog))))
				(define hierarchical_order
					(order_items_follow_join_tree? ordered_sources first_alias order_items stage_catalog final_condition))
				(define needed_exprs (merge (list
					(extract_assoc fields (lambda (_title expr) expr))
					(list final_condition)
					(order_exprs order_items)
					(source_join_exprs scan_sources))))
				(if (or direct_order_safe
					(and hierarchical_order
						(or (not (query_limit_active? (qb_offset block) (qb_limit block)))
							(driver_limit_cannot_brake?
								ordered_sources first_alias final_condition
								(qb_offset block) (qb_limit block) stage_catalog))))
					(build_join_scan_rows
						(qb_schema block) scan_sources scan_plan first_alias needed_exprs
						final_condition fields order_items (qb_offset block) (qb_limit block)
						direct_order_safe stage_catalog (qb_facts block))
					(if hierarchical_order
						(if (ordered_join_native_limit_supported?
							scan_sources scan_plan first_alias order_items stage_catalog final_condition)
							(join_ordered_streaming_limit_plan
								(qb_schema block) scan_sources scan_plan first_alias needed_exprs
								final_condition order_items (qb_offset block) (qb_limit block) stage_catalog
								(qb_facts block)
								(lambda (probe_work_rows scalar_probe)
									(cons (quote list) (lower_join_result_fields
										scan_sources first_alias
										(if (nil? scalar_probe) fields
											(rewrite_physical_scalar_probe_as_true scalar_probe fields))
										probe_work_rows)))
								(list (quote lambda) (list (quote _acc) (quote __ordered_join_row))
									(list (quote begin)
										(list (quote resultrow) (quote __ordered_join_row))
										(quote _acc)))
								nil)
							(neumann_fail "build_queryplan"
								"ordered variable-cardinality join requires a streaming consumer"))
						(if (physical_prejoin_supported? block)
							(lower_query_block_through_prejoin block)
							(neumann_fail "build_queryplan"
								"ORDER BY has no streamable driver in the logical join tree")))
))))))

(define zero_source_field_expr_key (lambda (expr)
	(serialize expr)))

(define zero_source_shared_field_symbol (lambda (key)
	(symbol (concat "__zero_source_field_" (fnv_hash key)))))

/* A source-free SELECT evaluates its projection once. Logical stage interning
can make hundreds of output aliases reference the same scalar marker; lower
that marker once as well instead of rebuilding the same physical recipe for
every title. */
(define lower_zero_source_result_expr (lambda (fields)
	(begin
		(define counts (reduce (extract_assoc fields (lambda (_title expr) expr))
			(lambda (result expr)
				(begin
					(define key (zero_source_field_expr_key expr))
					(set_assoc result key (+ (get_assoc result key 0) 1)))) '()))
		(define repeated (filter (extract_assoc counts (lambda (key count)
			(if (> count 1) key nil))) (lambda (key) (not (nil? key)))))
		(define definitions (map repeated (lambda (key)
			(begin
				(define expr (reduce (extract_assoc fields (lambda (_title candidate) candidate))
					(lambda (found candidate) (if (not (nil? found)) found
						(if (equal? (zero_source_field_expr_key candidate) key) candidate nil))) nil))
				(list (quote define) (zero_source_shared_field_symbol key)
					(lower_scalar_marker_expr expr))))))
		(define row (list (quote resultrow) (cons (quote list)
			(map_assoc fields (lambda (_title expr)
				(begin
					(define key (zero_source_field_expr_key expr))
					(if (> (get_assoc counts key 0) 1)
						(zero_source_shared_field_symbol key)
						(lower_scalar_marker_expr expr))))))))
		(if (empty_list? definitions) row
			(cons (quote !begin) (append definitions row))))))

(define lower_zero_source_query_block (lambda (block)
	(if (equal? (coalesceNil (qb_where block) true) true)
		(lower_zero_source_result_expr (qb_fields block))
		(list (quote if)
			(lower_scalar_marker_expr (qb_where block))
			(lower_zero_source_result_expr (qb_fields block))
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
			(quote +) 0 (quote +) nil))))

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
					(list (list (lower_group_stage_prepare_using (list stage) (list stage) stage true nil)) final_block nil)
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

(define prepare_call_key (lambda (expr)
	(if (and (list? expr) (equal? (count expr) 3))
		(begin
			(define head (nth expr 0))
			(define target (nth expr 1))
			(if (and
				(or (equal? head (quote apply)) (equal? head (symbol "apply")))
				(list? target)
				(equal? (count target) 2))
				(begin
					(define session_call (nth target 0))
					(if (and
						(list? session_call)
						(equal? (count session_call) 2)
						(or
							(equal? (nth session_call 0) (quote context))
							(equal? (nth session_call 0) (symbol "context")))
						(equal? (nth session_call 1) "session"))
						(nth target 1)
						nil))
				nil))
		nil)))

(define emitted_prepare_call_keys (lambda (expr keys)
	(begin
		(define key (prepare_call_key expr))
		(define found (if (nil? key) keys (set_assoc keys key true)))
		(match expr
			((symbol quote) _value) found
			((quote quote) _value) found
			(cons head tail) (reduce tail (lambda (acc item)
				(emitted_prepare_call_keys item acc))
				(emitted_prepare_call_keys head found))
			_ found))))

/* Carrier consolidation can retain calls through nested consumer paths after
their physical recipe has been merged into another owner. Keep those logical
handles callable without rebuilding the already consolidated carrier. */
(define complete_emitted_prepare_bindings (lambda (ir plan)
	(if (not (query_block? (ir_root ir)))
		plan
		(begin
			(define catalog (unique_stages_by_id
				(stage_catalog_with_nested (query_block_stage_catalog (ir_root ir)))))
			(define bound_keys (emitted_prepare_binding_keys plan '()))
			(define called_keys (emitted_prepare_call_keys plan '()))
			(define missing_roots (filter catalog (lambda (stage)
				(begin
					(define key (stage_prepare_key stage))
					(and (has_assoc? called_keys key) (not (has_assoc? bound_keys key)))))))
			(if (empty_list? missing_roots)
				plan
				(cons (quote !begin) (merge (list
					(map missing_roots prepared_stage_binding)
					(list plan)))))))))

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

(define prepare_recipe_helper_symbol (lambda (key)
	(symbol (concat "__prepare_recipe_" (fnv_hash key)))))

(define collect_prepare_binding_occurrences (lambda (expr entries)
	(begin
		(define key (prepare_binding_key expr))
		(define accumulated (if (nil? key) entries
			(begin
				(define previous (get_assoc entries key nil))
				(set_assoc entries key (list expr
					(if (nil? previous) 1 (+ (cadr previous) 1)))))))
		(match expr
			(cons head tail) (reduce tail (lambda (found item)
				(collect_prepare_binding_occurrences item found))
				(collect_prepare_binding_occurrences head accumulated))
			_ accumulated))))

(define rewrite_repeated_prepare_bindings (lambda (expr repeated_keys retained_key)
	(begin
		(define key (prepare_binding_key expr))
		(if (and (not (nil? key))
			(and (has_assoc? repeated_keys key) (not (equal? key retained_key))))
			(list (prepare_recipe_helper_symbol key))
			(match expr
				(cons head tail) (cons
					(rewrite_repeated_prepare_bindings head repeated_keys retained_key)
					(map tail (lambda (item)
						(rewrite_repeated_prepare_bindings item repeated_keys retained_key))))
				_ expr)))))

/* `once` makes repeated lazy prepares runtime-idempotent, but copying their
initializer into every projected field still makes recipe emission and
serialization proportional to initializer size times consumer count. Retain
the call sites (and therefore laziness/braking) while owning each repeated AST
recipe in one zero-argument helper. */
(define deduplicate_lazy_prepare_recipes (lambda (plan)
	(begin
		(define occurrences (collect_prepare_binding_occurrences plan '()))
		(define repeated (filter (extract_assoc occurrences (lambda (key entry)
			(if (> (cadr entry) 1) (list key (car entry)) nil)))
			(lambda (entry) (not (nil? entry)))))
		(if (empty_list? repeated)
			plan
			(begin
				(define repeated_keys (reduce repeated (lambda (keys entry)
					(set_assoc keys (car entry) true)) '()))
				(cons (quote !begin) (merge (list
					(map repeated (lambda (entry)
						(list (quote define)
							(prepare_recipe_helper_symbol (car entry))
							(list (quote lambda) '()
								(rewrite_repeated_prepare_bindings
									(cadr entry) repeated_keys (car entry))))))
					(list (rewrite_repeated_prepare_bindings plan repeated_keys nil))))))))))

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
						catalog stage true)
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

(define closed_group_prepare_consolidation_required? (lambda (ir)
	(and (query_block? (ir_root ir))
		(reduce
			(stage_catalog_with_nested
				(query_block_stage_catalog (ir_root ir)))
			(lambda (shared stage)
				(or shared (stage_shared_prepare? stage)))
			false))))

/* Recipe emission is a two-step physical pass: normal lowering records which
lazy stage keys are actually reachable, then this collector emits one closed
initializer owner per canonical carrier and replaces local copies with aliases.
Both AST walks are linear; no pairwise recipe comparison is performed. */
(define consolidate_closed_group_prepares (lambda (ir plan)
	(if (not (closed_group_prepare_consolidation_required? ir))
		plan
		(begin
			(define catalog (stage_catalog_with_nested
				(query_block_stage_catalog (ir_root ir))))
			/* Unique carriers already own exactly one initializer. The consolidation
			pass exists only for canonical backbones shared by multiple logical
			stages; without one, its repeated full-plan usage and rewrite walks are
			pure allocation overhead on wide scalar read models. */
			(begin
				(define emitted_keys (emitted_prepare_binding_keys plan '()))
				(define emitted_call_keys (emitted_prepare_call_keys plan '()))
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
								(or (has_assoc? emitted_call_keys (stage_prepare_key stage))
									(stage_aggregate_referenced? emitted_strings stage))))))))
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
								(list (without_prepare_bindings plan selected_keys))))))))))))

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
		(define complete_plan (complete_emitted_prepare_bindings ir consolidated_plan))
		(define deduplicated_plan (deduplicate_lazy_prepare_recipes complete_plan))
		(define memoized_plan (if (empty_list? (ir_stages ir))
			deduplicated_plan
			(consolidate_query_invariant_presence_memos deduplicated_plan)))
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
	(begin
		(context "check")
		(define ir (decorrelate_logical_query ast))
		(context "check")
		(define reordered (optimize_logical_query ir))
		(context "check")
		(define prepared (prepare_physical_queryplan reordered))
		(context "check")
		(define plan (emit_physical_queryplan prepared))
		(context "check")
		plan)))

(define neumann_compile_ir_pipeline (lambda (ir)
	(begin
		(context "check")
		(define normalized (require_flat_stage_dependencies "compile_ir" (normalize_stage_dependencies ir)))
		(context "check")
		(define reordered (optimize_logical_query normalized))
		(context "check")
		(define prepared (prepare_physical_queryplan reordered))
		(context "check")
		(define plan (emit_physical_queryplan prepared))
		(context "check")
		plan)))

/* ------------------------------------------------------------------------- */
/* Parser-facing adapters                                                     */

(define build_queryplan_term (lambda (query)
	(neumann_compile_pipeline query)))

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
			(or (physical_expr_has_head? head target)
				(reduce tail (lambda (found item)
					(or found (physical_expr_has_head? item target))) false)))
		_ false)))

(define physical_prefiltered_membership_expr? (lambda (expr)
	(match expr
		((symbol scan_recset) _tx source _cols _filter)
		(or (physical_expr_has_head? source (quote recset_project_join))
			(physical_prefiltered_membership_expr? source))
		((quote scan_recset) _tx source _cols _filter)
		(or (physical_expr_has_head? source (quote recset_project_join))
			(physical_prefiltered_membership_expr? source))
		(cons head tail) (or
			/* Prefiltered candidates are encoded as an immediately applied lambda:
			the body projects membership over the RecSet supplied by its argument.
			A list-valued call head is executable AST and must be traversed too. */
			(and (physical_expr_has_head? head (quote recset_project_join))
				(physical_expr_has_head? tail (quote scan_recset)))
			(or (physical_prefiltered_membership_expr? head)
				(reduce tail (lambda (found item)
					(or found (physical_prefiltered_membership_expr? item))) false)))
		_ false)))

(define physical_membership_operator_family (lambda (plan)
	(if (physical_expr_has_head? plan (quote scan_order_batch_accept))
		"ordered_batch_accept"
		(if (physical_prefiltered_membership_expr? plan)
			"prefiltered_candidate_keyset"
			/* Calibration compiles one membership decision at a time, but the query
			may contain unrelated projected RecSets (for example an ACL carrier). The
			key index uniquely identifies the selected ordered-driver implementation,
			so inspect it before the more general recset_project_join primitive. */
			(if (physical_expr_has_head? plan (quote recset_key_index))
				"driver_order_membership_probe"
				(if (physical_expr_has_head? plan (quote recset_project_join))
					"candidate_keyset"
					"driver_filter_join_probe"))))))

(define physical_operator_family_for_decision (lambda (plan decision)
	(begin
		(define kind (qassoc_get decision "decision" nil))
		(if (equal? kind "membership_carrier")
			(physical_membership_operator_family plan)
			(if (equal? kind "ordered_recset_consumer")
				(begin
					(define chosen (qassoc_get decision "chosen" nil))
					(if (or (equal? chosen "ordered_direct_recset")
						(equal? chosen "ordered_base_membership"))
						chosen
						"unknown"))
				"unknown")))))

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
		(define decision_kind (qassoc_get decision "decision" nil))
		(define expected_family (if (or (equal? decision_kind "membership_carrier")
			(equal? decision_kind "ordered_recset_consumer"))
			variant operator_family))
		(define consistent (equal? operator_family expected_family))
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
							"carrier_rows" (physical_calibration_input decision "carrier_rows")
							"candidate_density" (physical_calibration_input decision "candidate_density")
							"estimate_population" (physical_calibration_input decision "estimate_population")
							"estimate_coverage" (physical_calibration_input decision "estimate_coverage")
							"projected_driver_rows" (physical_calibration_input decision "projected_driver_rows")
							"driver_input_rows" (physical_calibration_input decision "driver_input_rows")
							"driver_rows" (physical_calibration_input decision "driver_rows")
							"prefiltered_driver_rows" (physical_calibration_input decision "prefiltered_driver_rows")
							"expected_driver_rows_visited" (physical_calibration_input decision "expected_driver_rows_visited")
							"limit" (physical_calibration_input decision "limit")
							"offset" (physical_calibration_input decision "offset")
							"probe_branches" (physical_calibration_input decision "probe_branches")
							"candidate_scan_invocations" (physical_calibration_input decision "candidate_scan_invocations")
							"candidate_filter_columns" (physical_calibration_input decision "candidate_filter_columns")
							"candidate_map_columns" (physical_calibration_input decision "candidate_map_columns")
							"candidate_cache_map_columns" (physical_calibration_input decision "candidate_cache_map_columns")
							"candidate_cache_backed" (physical_calibration_input decision "candidate_cache_backed")
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
					(physical_operator_family_for_decision
						(nth compilation 0) variant_decision)
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
						(physical_operator_family_for_decision
							(nth compilation 0) decision)
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
