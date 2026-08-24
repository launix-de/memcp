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

(define scalar_first_probe_outer_symbol? (lambda (sources expr)
	(and (equal? expr (symbol (string expr)))
		(reduce (coalesceNil sources '()) (lambda (found src)
			(or found
				(match (string expr)
					(concat alias "." _col) (equal? alias (source_alias src))
					_ false))) false))))

(define mark_scalar_first_probe_outer_symbols (lambda (sources expr)
	(match expr
		((symbol quote) _value) expr
		((quote quote) _value) expr
		((symbol outer) _value) expr
		((quote outer) _value) expr
		(cons head tail) (cons head (map tail (lambda (item)
			(mark_scalar_first_probe_outer_symbols sources item))))
		_ (if (scalar_first_probe_outer_symbol? sources expr)
			(list (quote outer) expr)
			expr))))

(define scalar_first_probe_key_terms (lambda (sources default_alias src keys lookup_keys)
	(begin
		(define alias (source_alias src))
		(map (produceN (count keys)) (lambda (i)
			(list (quote equal??)
				(lower_column_expr_for_alias src (nth keys i))
				(mark_scalar_first_probe_outer_symbols sources
					(lower_column_expr_for_join sources default_alias (nth lookup_keys i)))))))))

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
		/* Direct probes own this subtree even when the consumer did not need an
		explicit promotion marker. Keep the complete immutable catalog on every
		nested group so its prepare plan can resolve transitive stage-output
		sources before scanning their physical carriers. */
		(map direct_stages (lambda (nested_stage)
			(if (group_stage? nested_stage)
				(group_stage_with_stage_catalog nested_stage stage_lookup)
				nested_stage))))))

(define scalar_first_query_probe_nested_stages_using_index (lambda (direct_stages closure_index)
	(unique_stages_by_id (merge (list
		direct_stages
		(merge (map direct_stages (lambda (nested_stage)
			(get_assoc closure_index (logical_stage_key nested_stage))))))))))

(define lower_direct_scalar_query_probe (lambda (input value_expr partition_limit on_overflow)
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
				(define check_cardinality (equal? on_overflow (quote error)))
				(define raw_probe (list (quote scan_order)
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
					partition_limit
					(cons (quote list) mapcols)
					(list (quote lambda)
						(map mapcols (lambda (col) (symbol (concat (source_alias src) "." col))))
						(lower_column_expr_for_alias src value_expr))
					(if check_cardinality
						(scalar_query_probe_reduce_cardinality)
						(scalar_once_reduce_first))
					(if check_cardinality
						(list (quote quote) scalar_query_probe_empty)
						nil)
					false))
				(if check_cardinality
					(list
						(list (quote lambda) (list (quote __scalar_probe_result))
							(list (quote if)
								(list (quote and)
									(list (quote symbol?) (quote __scalar_probe_result))
									(list (quote equal?) (quote __scalar_probe_result)
										(list (quote quote) scalar_query_probe_empty)))
								nil
								(quote __scalar_probe_result)))
						raw_probe)
					raw_probe))
			nil))))

(define lower_scalar_first_query_probe_expr_using (lambda (stage value_expr keys lookup_keys nested_stages prepare_stages inline_presence_stages partition_limit)
	(begin
		(define input (gs_input stage))
		(define inline_presence_ids (stage_id_set inline_presence_stages))
		(define inline_presence_sources (filter (qb_sources input) (lambda (src)
			(and (stage_output_relation? (source_relation src))
				(has_assoc? inline_presence_ids (stage_output_relation_id (source_relation src)))))))
		(define inline_presence_aliases (map inline_presence_sources source_alias))
		(define keyed_terms (map (produceN (count keys)) (lambda (i)
			(list (query_key_term_alias (qb_sources input) (nth keys i))
				(list (quote equal??) (nth keys i) (nth lookup_keys i))))))
		/* Keep correlation predicates in the relational filter even when they can
		also annotate an inner source for pushdown. The first source is the physical
		driver and therefore has no consuming join edge on which its join expression
		could otherwise be evaluated. An outer-domain-only key labels the scalar
		partition but does not constrain an input row; its correlation is already
		represented by the input query. */
		(define where_key_terms (map
			(filter keyed_terms (lambda (term)
				(and (not (nil? (nth term 0)))
					(not (contains? inline_presence_aliases (nth term 0))))))
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
			partition_limit
			0
			(qb_hidden prepared_input)
			(qb_stages prepared_input)
			(qb_facts prepared_input)))
		(define direct_probe (if (empty_list? prepare_stages)
			(lower_direct_scalar_query_probe prepared_input probe_value_expr partition_limit
				(qassoc_get (gs_facts stage) (quote on_overflow) nil))
			nil))
		(define probe_expr (if (nil? direct_probe)
			(begin
				(define reduced (lower_query_block_as_dataset_reduce
					bounded_prepared_input
					(list "__value" probe_value_expr)
					(list (quote lambda) (list (quote __value)) (quote __value))
					(if (equal? (qassoc_get (gs_facts stage) (quote on_overflow) nil) (quote error))
						(scalar_query_probe_reduce_cardinality)
						(scalar_query_probe_reduce_first))
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

(define bounded_scalar_query_probe_inline_presence_stages (lambda (closure_index stage_catalog direct_stages probe_work_rows)
	(begin
		(define eligible_stages (filter direct_stages (lambda (nested_stage)
			(and (group_stage? nested_stage)
				(and (or (presence_probe_stage? nested_stage)
					(not (stage_shared_prepare? nested_stage)))
					(and (scalar_or_presence_probe_stage? nested_stage)
						(and (not (stage_has_residual_outer_refs? nested_stage))
							(stage_direct_probe_cost_preferred? nested_stage probe_work_rows))))))))
		(define one_dependency_chain (reduce direct_stages (lambda (found root)
			(or found
				(begin
					(define closure_ids (stage_id_set
						(get_assoc closure_index (logical_stage_key root))))
					(reduce direct_stages (lambda (covered candidate)
						(and covered (has_assoc? closure_ids (gs_id candidate)))) true)))) false))
		/* A scalar dependency closure is one physical choice. Mixing direct scalar
		children with prepared parents duplicates work, while independent sibling
		projections must retain their shared carrier instead of emitting one scan per
		output column. A 0:1 presence stage without a semantically equivalent sibling
		can own its closure and use scan_exists beside prepared scalar siblings.
		Equivalent presence stages retain their combined carrier instead of repeating
		their enclosing scalar probes, including correlated stages whose concrete
		aliases prevent eager preparation. */
		(if (and one_dependency_chain
			(equal? (count eligible_stages) (count direct_stages)))
			eligible_stages
			(begin
				(define eligible_presence_stages (filter eligible_stages presence_probe_stage?))
				(define semantic_signatures (stage_semantic_signature_index stage_catalog))
				(filter eligible_presence_stages (lambda (stage)
					(begin
						(define signature (get_assoc semantic_signatures (gs_id stage)))
						(equal? (count (filter stage_catalog (lambda (candidate)
							(and (group_stage? candidate)
								(equal? signature (get_assoc semantic_signatures (gs_id candidate))))))) 1)))))))))

(define lower_scalar_first_query_probe_expr (lambda (all_stages stage value_expr keys lookup_keys probe_work_rows fallback_probe_work_rows)
	(begin
		(define direct_stages (scalar_first_query_probe_direct_nested_stages all_stages stage))
		(define probe_catalog (stage_catalog_with_nested
			(merge (list all_stages direct_stages))))
		(define dependency_graph (stage_dependency_graph probe_catalog))
		(define closure_index (stage_dependency_closure_index_using_graph dependency_graph direct_stages))
		(define nested_stages
			(scalar_first_query_probe_nested_stages_using_index direct_stages closure_index))
		(define fallback_probe_literal (planner_literal_value fallback_probe_work_rows))
		(define decision_probe_work_rows (if (number? (planner_literal_value probe_work_rows))
			probe_work_rows
			(if (and (equal? (count (gs_aggregates stage)) 1)
				(and (> (count (stage_dependency_closure_using_graph dependency_graph stage)) 1)
					(and (number? fallback_probe_literal) (<= fallback_probe_literal 1))))
				fallback_probe_work_rows
				probe_work_rows)))
		/* A bounded parent probe evaluates this subtree only for rows that survived
		root braking. Compare those expected probe calls with the dependent stage's
		input size; retain the group cache when repeated probes amortize its build. */
		(define inline_presence_stages (if (number? (planner_literal_value decision_probe_work_rows))
			(bounded_scalar_query_probe_inline_presence_stages closure_index probe_catalog direct_stages decision_probe_work_rows)
			'()))
		/* Once a parent is selected for direct probing, its complete dependency
		closure is owned by that probe. Preparing children separately would pay
		the carrier build cost in addition to the selected direct path. */
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
			(stage_partition_limit stage)))))

(define lower_scalar_aggregate_query_probe_expr (lambda (all_stages stage value_expr keys lookup_keys reduce_expr neutral_expr)
	(begin
		(define input (gs_input stage))
		(define keyed_terms (map (produceN (count keys)) (lambda (i)
			(list (query_key_term_alias (qb_sources input) (nth keys i))
				(list (quote equal??) (nth keys i) (nth lookup_keys i))))))
		(define where_key_terms (map
			(filter keyed_terms (lambda (term) (not (nil? (nth term 0)))))
			(lambda (term) (nth term 1))))
		(define keyed_input (make_query_block
			(qb_schema input)
			(query_sources_with_key_terms (qb_sources input) keyed_terms)
			(qb_fields input)
			(combine_where_terms
				(cons (qb_where input) where_key_terms)
				true)
			(qb_group input)
			(qb_having input)
			(qb_order input)
			(qb_limit input)
			(qb_offset input)
			(qb_hidden input)
			(qb_stages input)
			(qb_facts input)))
		(define nested_stages (scalar_first_query_probe_direct_nested_stages all_stages stage))
		(define prepared_input
			(query_block_without_stages_after_eager_prepare_using nested_stages keyed_input))
		(define probe_expr (lower_query_block_as_dataset_reduce
			prepared_input
			(list "__value" value_expr)
			(list (quote lambda) (list (quote __value)) (quote __value))
			reduce_expr
			neutral_expr
			nil))
		(if (empty_list? nested_stages)
			probe_expr
			(cons (quote !begin)
				(merge (list
					(lazy_stage_prepare_bindings nested_stages (filter nested_stages group_stage?))
					(map nested_stages (lambda (nested_stage)
						(if (group_stage? nested_stage)
							(stage_prepare_call_expr nested_stage)
							(lower_stage_prepare_using nested_stages nested_stages nested_stage))))
					(lower_stage_materialize_all nested_stages)
					(list probe_expr))))))))

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
						(stage_partition_limit bound_stage))))
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
		(define stage_catalog (unique_stages_by_id (merge (map (coalesceNil plans '()) (lambda (plan)
			(match plan
				'(_stage _requested_col nested_stages _hoisted_stages _prepare_stages _inline_presence_stages _bounded_consumer) nested_stages
				_ '()))))))
		(define shared_stages (filter stages stage_shared_prepare?))
		(define direct_stages (filter stages (lambda (stage) (not (stage_shared_prepare? stage)))))
		(merge (list
			(lazy_stage_prepare_bindings stage_catalog shared_stages)
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

(define lower_keytable_scalar_first_probe_expr (lambda (all_stages stage requested_col resolved_lookup_key partition_limit)
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
				partition_limit
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

(define scalar_first_probe_recset_source_parts (lambda (all_stages stage requested_col)
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
		(list
			(lower_recset_stage_prepare_once_expr stage_catalog physical_stage)
			(list (quote scan_recset)
				(list (quote session) "__memcp_tx")
				(list (quote table) cache_schema cache_relation)
				(quoted_runtime_list (list requested_col))
				(list (quote lambda) (list (symbol requested_col))
					(list (quote equal??) (symbol requested_col) true)))))))

(define lower_recset_scalar_first_probe_expr (lambda (all_stages stage requested_col resolved_lookup_key)
	(begin
		(define source_parts
			(scalar_first_probe_recset_source_parts all_stages stage requested_col))
		(define lookup_key (recset_scalar_first_probe_lookup_key stage))
		(list (quote begin)
			(car source_parts)
			(list (quote apply)
				(list
					(physical_query_session_symbol)
					"get_or_compute_scoped"
					(physical_query_scope_symbol)
					lookup_key
					(list (quote lambda) '()
						(list (quote recset_key_index)
							(list (quote session) "__memcp_tx")
							(cadr source_parts)
							(quoted_runtime_list (list "k0")))))
				(list (quote list) resolved_lookup_key))))))

/* Once the consuming scan has selected the RecSet alternative, project the
true group keys directly onto that scan's base-table record positions. This
is the carrier itself, not another membership decision: the group-stage
preparation, truth RecSet, and key projection all belong to the same physical
choice at the consuming join edge. */
(define lower_projected_recset_scalar_first_probe_expr (lambda (all_stages stage requested_col target_src target_col)
	(begin
		(define source_parts
			(scalar_first_probe_recset_source_parts all_stages stage requested_col))
		(list (quote begin)
			(car source_parts)
			(list (quote recset_project_join)
				(list (quote session) "__memcp_tx")
				(cadr source_parts)
				(quoted_runtime_list (list "k0"))
				(source_table_expr target_src)
				(quoted_runtime_list (list target_col)))))))

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

(define lower_table_scalar_first_probe_expr (lambda (sources default_alias src stage value_expr keys lookup_keys order_exprs dirs offset_value partition_limit)
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
			partition_limit
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
		(define partition_limit (stage_partition_limit stage))
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
					partition_limit))
			(symbol query-scan)
			(lower_scalar_first_query_probe_expr
				probe_stages
				stage
				value_expr
				keys
				(map lookup_keys (lambda (key) (lower_column_expr_for_join sources default_alias key)))
				probe_work_rows
				effective_probe_work_rows)
			(symbol table-scan)
			(if (presence_probe_stage? stage)
				/* EXISTS is a 0:1 relational fact. Its table-scan implementation can
				stop at the stage's partition limit without materializing an aggregate
				row; retain the scalar marker's historical 1/nil value contract for
				value-producing parents. */
				(list (quote if)
					(lower_driver_membership_probe_expr
						sources default_alias stage
						(if (empty_list? lookup_keys) nil (car lookup_keys)))
					1 nil)
				(lower_table_scalar_first_probe_expr
					sources default_alias src stage value_expr keys lookup_keys
					order_exprs dirs offset_value partition_limit))
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
		(if (query_block? src)
			(lower_scalar_aggregate_query_probe_expr
				(stage_catalog_with_nested (merge (list
					(qassoc_get (gs_facts stage) (quote probe_catalog) '())
					(if (lowering_catalog? (group_stage_lowering_catalog stage))
						(lowering_catalog_stages (group_stage_lowering_catalog stage))
						'())
					(list stage))))
				stage
				value_expr
				keys
				(map lookup_keys (lambda (key)
					(lower_column_expr_for_join sources default_alias key)))
				reduce_expr
				neutral_expr)
			(begin
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
					false))))))

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
		(if (query_block? src)
			(lower_scalar_first_query_probe_expr
				(stage_catalog_with_nested (merge (list
					(qassoc_get (gs_facts stage) (quote probe_catalog) '())
					(if (lowering_catalog? (group_stage_lowering_catalog stage))
						(lowering_catalog_stages (group_stage_lowering_catalog stage))
						'())
					(list stage))))
				stage
				value_expr
				keys
				(map lookup_keys (lambda (key)
					(lower_column_expr_for_join sources default_alias key)))
				(probe_context_row_count sources))
			(begin
				(define condition_cols (extract_columns_for_alias src condition))
				(define key_cols (merge_unique (map keys (lambda (expr) (extract_columns_for_alias src expr)))))
				(define value_cols (extract_columns_for_alias src value_expr))
				(define filtercols (merge_unique (list condition_cols key_cols)))
				(define unset (list (quote quote) (quote __scalar_cardinality_unset)))
				(define partition_limit (stage_partition_limit stage))
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
					partition_limit
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
					nil))))))

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
			(join_optimizer_at_most_one_unbound_source? stages src)
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

(define order_items_follow_join_tree_acc? (lambda (all_sources sources default_alias order_items stages condition bound_sources)
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
					(and (or
						(constant_scalar_or_presence_stage_output_source? stages (car sources))
						(source_unique_point_condition? (car sources)
							(combine_where condition (source_join_expr (car sources))))
						(and (not (empty_list? bound_sources))
							(source_is_unique_lookup_from_sources?
								all_sources default_alias bound_sources (car sources) stages condition)))
						(order_items_follow_join_tree_acc? all_sources (cdr sources) default_alias order_items stages condition
							(cons (car sources) bound_sources)))
					(order_items_follow_join_tree_acc? all_sources (cdr sources) default_alias remaining stages condition
						(cons (car sources) bound_sources))))))))

(define order_items_follow_join_tree? (lambda (sources default_alias order_items stages condition)
	(order_items_follow_join_tree_acc? sources sources default_alias order_items stages condition '())))

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

(define candidate_recset_local_condition (lambda (sources default_alias alias terms include_constants)
	(combine_where_terms
		(filter terms (lambda (term)
			(begin
				(define aliases (join_hypergraph_expr_aliases
					default_alias (source_aliases sources) term))
				(or (and include_constants (empty_list? aliases))
					(and (single_source? aliases) (equal? (car aliases) alias))))))
		true)))

(define candidate_recset_edge_columns (lambda (sources default_alias from_src to_src terms)
	(reduce terms (lambda (columns term)
		(begin
			(define aliases (join_hypergraph_expr_aliases
				default_alias (source_aliases sources) term))
			(if (and (equal? (count aliases) 2)
				(and (contains? aliases (source_alias from_src))
					(contains? aliases (source_alias to_src))))
				(match term
					'(op left right) (if (or (equal? op (quote equal?)) (equal? op (quote equal??)))
						(begin
							(define from_left (direct_column_name_for_alias from_src left))
							(define to_right (direct_column_name_for_alias to_src right))
							(define from_right (direct_column_name_for_alias from_src right))
							(define to_left (direct_column_name_for_alias to_src left))
							(if (and (not (nil? from_left)) (not (nil? to_right)))
								(list (append (car columns) from_left) (append (cadr columns) to_right))
								(if (and (not (nil? from_right)) (not (nil? to_left)))
									(list (append (car columns) from_right) (append (cadr columns) to_left))
									columns)))
						columns)
					_ columns)
				columns))) (list '() '()))))

(define candidate_recset_filter_source (lambda (src input condition)
	(begin
		(define alias (source_alias src))
		(define cols (extract_columns_for_alias src condition))
		(list (quote scan_recset)
			'(session "__memcp_tx")
			input
			(cons (quote list) cols)
			(list (quote lambda)
				(map cols (lambda (col) (scan_callback_symbol_for_alias alias col)))
				(lower_column_expr_for_alias src condition))))))

(define candidate_recset_join_chain (lambda (sources default_alias remaining terms current_src current_recset)
	(match remaining
		(cons next_src rest) (begin
			(define edge_columns (candidate_recset_edge_columns
				sources default_alias current_src next_src terms))
			(if (empty_list? (car edge_columns))
				nil
				(begin
					(define projected (list (quote recset_project_join)
						'(session "__memcp_tx")
						current_recset
						(quoted_runtime_list (car edge_columns))
						(source_table_expr next_src)
						(quoted_runtime_list (cadr edge_columns))))
					(define local_condition (candidate_recset_local_condition
						sources default_alias (source_alias next_src) terms false))
					(candidate_recset_join_chain sources default_alias rest terms next_src
						(candidate_recset_filter_source next_src projected local_condition)))))
		_ (list current_src current_recset))))

(define joined_candidate_recset_branch_carrier (lambda (branch)
	(if (not (candidate_recset_branch_supported? branch))
		nil
		(begin
			(define sources (qb_sources branch))
			(define tree (query_block_join_plan branch sources))
			(define ordered_sources (join_optimizer_sources_for_order
				sources (join_optimizer_tree_aliases tree)))
			(define first_src (car ordered_sources))
			(define default_alias (qassoc_get (qb_facts branch)
				(quote default_alias) (source_alias (car sources))))
			(define terms (candidate_recset_branch_terms branch tree))
			(define initial_condition (candidate_recset_local_condition
				sources default_alias (source_alias first_src) terms true))
			(define initial_recset (candidate_recset_filter_source
				first_src (source_table_expr first_src) initial_condition))
			(define joined (candidate_recset_join_chain sources default_alias
				(cdr ordered_sources) terms first_src initial_recset))
			(if (or (nil? joined) (nil? (cadr joined)))
				nil
				(begin
					(define output_src (car joined))
					(define output_col (direct_column_name_for_alias
						output_src (query_block_first_expr branch)))
					(if (nil? output_col) nil
						(list output_src output_col (cadr joined)))))))))

(define candidate_recset_branch_carrier (lambda (branch)
	(if (> (count (qb_sources branch)) 1)
		(joined_candidate_recset_branch_carrier branch)
		(match (recset_project_join_branch_parts branch)
			'(src source_col) (begin
				(define alias (source_alias src))
				(define condition (combine_where (qb_where branch) (source_join_expr src)))
				(define filtercols (extract_columns_for_alias src condition))
				(list src source_col
					(list (quote scan_recset)
						'(session "__memcp_tx")
						(source_table_expr src)
						(cons (quote list) filtercols)
						(list (quote lambda)
							(map filtercols (lambda (col) (symbol (concat alias "." col))))
							(lower_column_expr_for_alias src condition)))))
			_ nil))))

(define joined_candidate_recset_branch_expr (lambda (target_src branch target_col)
	(begin
		(define carrier (candidate_recset_branch_carrier branch))
		(if (nil? carrier)
			nil
			(list (quote recset_project_join)
				'(session "__memcp_tx")
				(nth carrier 2)
				(quoted_runtime_list (list (nth carrier 1)))
				(source_table_expr target_src)
				(quoted_runtime_list (list target_col)))))))

(define recset_project_join_branch_expr (lambda (target_src branch target_col)
	(if (> (count (qb_sources branch)) 1)
		(joined_candidate_recset_branch_expr target_src branch target_col)
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
			_ nil))))

/* Build the exact, query-local membership subset for one ordered driver batch.
The forward projection limits the candidate relation before its predicate is
evaluated; the reverse projection returns matching keys to the driver table.
The final intersection is required because a repeated driver key may project
rows outside the current batch. */
(define batch_membership_branch_expr (lambda (target_src branch target_col batch_expr)
	(match (recset_project_join_branch_parts branch)
		'(src source_col) (begin
			(define alias (source_alias src))
			(define condition (combine_where (qb_where branch) (source_join_expr src)))
			(define filtercols (extract_columns_for_alias src condition))
			(define source_candidates (list (quote recset_project_join)
				'(session "__memcp_tx")
				batch_expr
				(quoted_runtime_list (list target_col))
				(source_table_expr src)
				(quoted_runtime_list (list source_col))))
			(define source_matches (list (quote scan_recset)
				'(session "__memcp_tx")
				source_candidates
				(cons (quote list) filtercols)
				(list (quote lambda)
					(map filtercols (lambda (col) (symbol (concat alias "." col))))
					(lower_column_expr_for_alias src condition))))
			(list (quote recset_intersect)
				(cons (quote list) (list
					batch_expr
					(list (quote recset_project_join)
						'(session "__memcp_tx")
						source_matches
						(quoted_runtime_list (list source_col))
						(source_table_expr target_src)
						(quoted_runtime_list (list target_col)))))))
		_ nil)))

(define batch_membership_base_expr (lambda (target_src stage target_col batch_expr)
	(begin
		(define descriptor (membership_keyset_descriptor (list stage nil target_col nil)))
		(if (nil? descriptor)
			nil
			(begin
				(define src (nth descriptor 0))
				(define source_col (nth descriptor 1))
				(define condition (nth descriptor 2))
				(define alias (source_alias src))
				(define filtercols (extract_columns_for_alias src condition))
				(define source_candidates (list (quote recset_project_join)
					'(session "__memcp_tx")
					batch_expr
					(quoted_runtime_list (list target_col))
					(source_table_expr src)
					(quoted_runtime_list (list source_col))))
				(define source_matches (list (quote scan_recset)
					'(session "__memcp_tx")
					source_candidates
					(cons (quote list) filtercols)
					(list (quote lambda)
						(map filtercols (lambda (col) (symbol (concat alias "." col))))
						(lower_column_expr_for_alias src condition))))
				(list (quote recset_intersect)
					(cons (quote list) (list
						batch_expr
						(list (quote recset_project_join)
							'(session "__memcp_tx")
							source_matches
							(quoted_runtime_list (list source_col))
							(source_table_expr target_src)
							(quoted_runtime_list (list target_col)))))))))))

(define batch_membership_expr (lambda (target_src membership batch_expr)
	(begin
		(define stage (nth membership 0))
		(define target_col (nth membership 2))
		(define input (gs_input stage))
		(if (union_block? input)
			(begin
				(define branches (map (union_branches input) (lambda (branch)
					(batch_membership_branch_expr target_src branch target_col batch_expr))))
				(if (reduce branches (lambda (unsupported branch_expr)
					(or unsupported (nil? branch_expr))) false)
					nil
					(list (quote recset_intersect)
						(cons (quote list) (list
							batch_expr
							(if (single_source? branches)
								(car branches)
								(list (quote recset_union) (cons (quote list) branches))))))))
			(batch_membership_base_expr target_src stage target_col batch_expr)))))

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

/* Presence relations cache a raw count (consumers wrap it in ">0"); scalar
relations cache the literal boolean value. This decision follows relational
NULL semantics, never the syntactic reason that introduced the stage. */
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
(define recset_project_join_plan_for_membership_using (lambda (src membership consumer driver_rows_override allow_ordered_batch prefiltered_driver_expr)
	(begin
		(define stage (nth membership 0))
		/* Some late RecSet consumers are introduced after reorder telemetry was
		attached. Reconstruct only the candidate's scalar work from the existing
		logical stage; this is one formula walk, never an alternative plan build. */
		(define facts (merge (list (membership_candidate_work_facts stage) (gs_facts stage))))
		(define cost_facts (if (equal? consumer (quote aggregate))
			(qassoc_set facts (quote membership_consumer) (quote aggregate))
			facts))
		(define driver_probe_supported (membership_driver_subscan_supported? stage))
		(define raw_expr (recset_project_join_expr_for_membership_raw src membership))
		(if (or (nil? raw_expr)
			(not (or
				(equal? (qassoc_get facts (quote purpose) nil) (quote in_membership))
				(equal? (qassoc_get facts (quote purpose) nil) (quote in_candidate)))))
			(if (nil? raw_expr) nil (list "candidate_keyset" raw_expr))
			(begin
				/* Membership reorder already records this estimate in stage facts. Only
				measure here for callers that reached physical lowering without those
				facts; repeating a selective estimate may build an entire adaptive text
				index even though the sample itself is capped. */
				(define measured_estimate (if (nil? (qassoc_get facts (quote membership_candidate_estimated_rows) nil))
					(planner_stage_filter_estimate (gs_input stage) 512)
					nil))
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
					(qassoc_get facts (quote membership_candidate_matching_rows) nil)
					(qassoc_get measured_estimate (quote rows) nil)))
				(define costed_candidate_rows
					(qassoc_get facts (quote membership_candidate_estimated_rows) nil))
				/* Reorder owns stage cardinality. Physical lowering consumes that
				estimate verbatim; re-extrapolating the merged UNION sample here was a
				second, contradictory planner that could turn a selective text path
				back into 100% of its input. */
				(define candidate_rows (coalesceNil costed_candidate_rows
					(planner_estimated_matching_rows
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
						candidate_input_rows candidate_input_rows)))
				(define source_rows (planner_source_row_count src))
				/* Ordered LIMIT consumes only its local window before downstream
				operators. Cost this edge with that workload instead of a global
				driver cardinality; it is the relevant side of the plan inequality. */
				(define driver_rows (coalesceNil driver_rows_override
					(coalesceNil
						(qassoc_get facts (quote membership_driver_rows) nil)
						source_rows)))
				(define decision_id (concat "membership_carrier:" (gs_id stage)))
				(define known (and (number? candidate_rows) (number? driver_rows)))
				(define owns_requirement (qassoc_get facts (quote physical_membership_requirement) false))
				/* A membership below OR cannot become the scan driver without
				dropping sibling-accepted rows. For a broad ordered window retain
				the lazy probe carrier; the ordinary cost inequality owns every
				feasible carrier pair outside this semantic/Top-K constraint. */
				(define guarded_broad_order_driver
					(qassoc_get facts (quote guarded_broad_order_driver) false))
				(define candidate_cost (if known
					(membership_projection_cost candidate_input_rows candidate_rows driver_rows cost_facts)
					nil))
				(define driver_cost (if (and known driver_probe_supported)
					(membership_ordered_driver_probe_cost candidate_input_rows candidate_rows driver_rows cost_facts)
					nil))
				(define batch_cost_facts (if allow_ordered_batch
					(merge (list
						(list
							(list (quote membership_candidate_input_rows) candidate_input_rows)
							(list (quote membership_candidate_estimated_rows) candidate_rows)
							(list (quote membership_driver_rows) driver_rows)
							(list (quote membership_driver_input_rows) source_rows))
						cost_facts)) nil))
				(define batch_cost (if (and allow_ordered_batch known)
					(ordered_batch_accept_cost batch_cost_facts)
					nil))
				(define prefiltered_driver_rows
					(qassoc_get facts (quote membership_driver_rows) nil))
				(define prefiltered_driver_var (symbol "__prefiltered_membership_recset"))
				(define prefiltered_body (if (nil? prefiltered_driver_expr)
					nil
					(batch_membership_expr src membership prefiltered_driver_var)))
				(define prefiltered_expr (if (nil? prefiltered_body)
					nil
					(list
						(list (quote lambda) (list prefiltered_driver_var) prefiltered_body)
						prefiltered_driver_expr)))
				(define prefiltered_eligible (and
					(not (nil? prefiltered_expr))
					(and (number? prefiltered_driver_rows)
						(and (number? source_rows) (< prefiltered_driver_rows source_rows)))))
				(define prefiltered_cost (if (and prefiltered_eligible (number? candidate_rows))
					(membership_prefiltered_candidate_cost
						candidate_input_rows candidate_rows prefiltered_driver_rows cost_facts)
					nil))
				(define cost_choices (filter (list
					(if (nil? candidate_cost) nil (list "candidate_keyset" candidate_cost))
					(if (nil? driver_cost) nil (list "driver_order_membership_probe" driver_cost))
					(if (nil? batch_cost) nil (list "ordered_batch_accept" batch_cost))
					(if (nil? prefiltered_cost) nil
						(list "prefiltered_candidate_keyset" prefiltered_cost)))
					(lambda (choice) (not (nil? choice)))))
				(define cost_choice (if (empty_list? cost_choices)
					nil
					(reduce (cdr cost_choices) (lambda (best choice)
						(if (planner_cost_better? (cadr choice) (cadr best)) choice best))
						(car cost_choices))))
				(define normal_choice (if (and guarded_broad_order_driver driver_probe_supported)
					"driver_order_membership_probe"
					(if (nil? cost_choice)
						(if (and owns_requirement driver_probe_supported)
							"driver_order_membership_probe" "candidate_keyset")
						(car cost_choice))))
				(define alternatives (merge (list
					(cons "candidate_keyset"
						(if driver_probe_supported (list "driver_order_membership_probe") '()))
					(if allow_ordered_batch (list "ordered_batch_accept") '())
					(if prefiltered_eligible (list "prefiltered_candidate_keyset") '()))))
				(define chosen (planner_physical_choice decision_id normal_choice alternatives))
				(define forced (planner_physical_override decision_id))
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
							(list "driver_input_rows" source_rows)
							(list "driver_rows" driver_rows)
							(list "prefiltered_driver_rows" prefiltered_driver_rows)
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
						(list "alternatives" (filter (list
							(list
								(list "plan" "candidate_keyset")
								(list "status" (if (equal? chosen "candidate_keyset") "chosen" "rejected"))
								(list "reason" (if (equal? chosen "candidate_keyset") "selected" (if known "higher_total_ns_or_forced_alternative" "unknown_statistics_fallback")))
								(list "cost" (if (nil? candidate_cost) '() (planner_cost_explain candidate_cost))))
							(list
								(list "plan" "driver_order_membership_probe")
								(list "status" (if (equal? chosen "driver_order_membership_probe") "chosen" "rejected"))
								(list "reason" (if (equal? chosen "driver_order_membership_probe") "selected" "higher_total_ns_or_forced_alternative"))
								(list "cost" (if (nil? driver_cost) '() (planner_cost_explain driver_cost))))
							(if allow_ordered_batch
								(list
									(list "plan" "ordered_batch_accept")
									(list "status" (if (equal? chosen "ordered_batch_accept") "chosen" "rejected"))
									(list "reason" (if (equal? chosen "ordered_batch_accept") "selected" "higher_total_ns_or_forced_alternative"))
									(list "cost" (if (nil? batch_cost) '() (planner_cost_explain batch_cost))))
								nil)
							(if prefiltered_eligible
								(list
									(list "plan" "prefiltered_candidate_keyset")
									(list "status" (if (equal? chosen "prefiltered_candidate_keyset") "chosen" "rejected"))
									(list "reason" (if (equal? chosen "prefiltered_candidate_keyset") "selected" "higher_total_ns_or_forced_alternative"))
									(list "cost" (planner_cost_explain prefiltered_cost)))
								nil)) (lambda (alternative) (not (nil? alternative)))))))
				(list chosen (if (equal? chosen "prefiltered_candidate_keyset")
					prefiltered_expr raw_expr)))))))

/* General expression callers preserve the established contract: only a
candidate-keyset choice replaces the marker with a projected RecSet carrier. */
(define recset_project_join_expr_for_membership_using (lambda (src membership consumer driver_rows_override allow_ordered_batch)
	(begin
		(define plan (recset_project_join_plan_for_membership_using
			src membership consumer driver_rows_override allow_ordered_batch nil))
		(if (and (not (nil? plan)) (equal? (car plan) "candidate_keyset"))
			(cadr plan)
			nil))))

(define recset_project_join_expr_for_membership (lambda (src membership)
	(recset_project_join_expr_for_membership_using src membership nil nil false)))

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
		((symbol count_distinct) _expr) true
		((quote count_distinct) _expr) true
		((symbol group_concat_distinct) _expr _separator) true
		((quote group_concat_distinct) _expr _separator) true
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
		((symbol group_concat_distinct) agg_expr _separator) (list (count_distinct_descriptor agg_expr))
		((quote group_concat_distinct) agg_expr _separator) (list (count_distinct_descriptor agg_expr))
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
		((symbol group_concat_distinct) _agg_expr _separator) '()
		((quote group_concat_distinct) _agg_expr _separator) '()
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

(define group_concat_distinct_value_expr (lambda (read_expr separator)
	(list (quote if)
		(list (quote list?) read_expr)
		(list (quote reduce)
			(list (quote filter) read_expr
				(list (quote lambda) (list (quote value))
					(list (quote not) (list (quote nil?) (quote value)))))
			(list (quote lambda) (list (quote joined) (quote value))
				(list (quote if)
					(list (quote nil?) (quote joined))
					(list (quote concat) (quote value))
					(list (quote concat) (quote joined) separator (quote value))))
			nil)
		(list (quote if) (list (quote nil?) read_expr) nil (list (quote concat) read_expr)))))

(define group_concat_distinct_read_expr (lambda (input grouptbl agg_expr separator)
	(group_concat_distinct_value_expr
		(list (quote get_column) grouptbl false (aggregate_col_name_using input (count_distinct_descriptor agg_expr)) false)
		separator)))

(define direct_group_concat_distinct_read_expr (lambda (agg_expr separator)
	(group_concat_distinct_value_expr
		(list (quote get_assoc) (quote rowassoc) (aggregate_col_name (count_distinct_descriptor agg_expr)))
		separator)))

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
				((symbol group_concat_distinct) agg_expr separator)
				(group_concat_distinct_read_expr input grouptbl agg_expr separator)
				((quote group_concat_distinct) agg_expr separator)
				(group_concat_distinct_read_expr input grouptbl agg_expr separator)
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
				((symbol group_concat_distinct) agg_expr separator)
				(group_concat_distinct_read_expr input grouptbl agg_expr separator)
				((quote group_concat_distinct) agg_expr separator)
				(group_concat_distinct_read_expr input grouptbl agg_expr separator)
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
				((symbol group_concat_distinct) agg_expr separator)
				(direct_group_concat_distinct_read_expr agg_expr separator)
				((quote group_concat_distinct) agg_expr separator)
				(direct_group_concat_distinct_read_expr agg_expr separator)
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
		(and (group_stage? stage)
			(and (or (scalar_value_stage? stage) (presence_probe_stage? stage))
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
			(if (not (group_stage? stage))
				nil
				(begin
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
											(qassoc_set
												(gs_facts stage)
												(quote null_semantics)
												(quote aggregate))
											(quote partition_by) (gs_keys stage))
										(quote result_max_rows_per_partition) 1)
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
								(qassoc_set facts (quote condition) condition)))))))))))

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
				/* A scalar cardinality stage owns its bounded query input and overflow
				contract. A direct aggregate alternative may be costed for ordinary
				groups, but must not replace that semantic operator at the consumer. */
				(define stage (if (or (scalar_cardinality_probe_stage? original)
					(scalar_aggregate_probe_stage? original))
					original
					(coalesceNil direct original)))
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
(define planner_ordered_batch_accept_startup_ns 1)
/* END GENERATED COST CONSTANTS */
