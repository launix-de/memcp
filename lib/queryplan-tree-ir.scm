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


/* Paper-aligned tree IR adapter for flat scalar-subquery tuples.
Node forms:
;; (op-select  predicate child)
;; (op-map     projections child)
;; (op-groupby keys aggs having child)
;; (op-window  partition order computations child)
;; (op-join    type predicate left right)
;; (op-scan    schema table)
;;
`op-scan` keeps the current table descriptor payload in its `table` slot so the
existing flat tuple can round-trip without semantic changes while later work
ports the actual operator rules to the tree representation. */
(define planner_tree_ir_node_kind (lambda (node) (match node
	(cons sym _) (match sym
		(symbol op-select) (quote op-select)
		'op-select (quote op-select)
		'(quote op-select) (quote op-select)
		(symbol op-map) (quote op-map)
		'op-map (quote op-map)
		'(quote op-map) (quote op-map)
		(symbol op-groupby) (quote op-groupby)
		'op-groupby (quote op-groupby)
		'(quote op-groupby) (quote op-groupby)
		(symbol op-window) (quote op-window)
		'op-window (quote op-window)
		'(quote op-window) (quote op-window)
		(symbol op-join) (quote op-join)
		'op-join (quote op-join)
		'(quote op-join) (quote op-join)
		(symbol op-scan) (quote op-scan)
		'op-scan (quote op-scan)
		'(quote op-scan) (quote op-scan)
		(symbol op-dep-join) (quote op-dep-join)
		'op-dep-join (quote op-dep-join)
		'(quote op-dep-join) (quote op-dep-join)
		nil)
	nil)))
(define planner_tree_ir_scan (lambda (schema table)
	(list (quote op-scan) schema table)))
(define planner_tree_ir_join (lambda (join_type predicate left right)
	(list (quote op-join) join_type predicate left right)))
(define planner_tree_ir_dep_join (lambda (predicate left right accessing)
	(list (quote op-dep-join) predicate left right accessing)))
(define planner_tree_ir_select (lambda (predicate child)
	(list (quote op-select) predicate child)))
(define planner_tree_ir_map (lambda (projections child)
	(list (quote op-map) projections child)))
(define planner_tree_ir_groupby (lambda (keys aggs having child)
	(list (quote op-groupby) keys aggs having child)))
(define planner_tree_ir_window (lambda (partition order computations child)
	(list (quote op-window) partition order computations child)))
(define planner_tree_ir_window_with_partition (lambda (tree partition)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-window))
		(planner_tree_ir_window
			partition
			(planner_tree_ir_window_order tree)
			(nth tree 3)
			(planner_tree_ir_window_child tree))
		tree)))
(define planner_tree_ir_window_computations (lambda (limit offset)
	(list
		(list (quote limit) limit)
		(list (quote offset) offset))))
(define planner_tree_ir_lookup_computation (lambda (computations key)
	(reduce (coalesceNil computations '()) (lambda (found entry)
		(if (not (nil? found))
			found
			(match entry
				'(entry_key entry_value) (if (equal? entry_key key) entry_value nil)
				nil)))
		nil)))
(define planner_tree_ir_window_order (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-window))
		(nth tree 2)
		nil)))
(define planner_tree_ir_window_partition (lambda (tree)
	(match tree
		'((quote op-window) partition _ _ _) partition
		_ nil)))
(define planner_tree_ir_window_limit (lambda (tree)
	(planner_tree_ir_lookup_computation
		(if (equal? (planner_tree_ir_node_kind tree) (quote op-window))
			(nth tree 3)
			'())
		(quote limit))))
(define planner_tree_ir_window_offset (lambda (tree)
	(planner_tree_ir_lookup_computation
		(if (equal? (planner_tree_ir_node_kind tree) (quote op-window))
			(nth tree 3)
			'())
		(quote offset))))
(define planner_tree_ir_window_child (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-window))
		(nth tree 4)
		nil)))
(define planner_tree_ir_dep_join_predicate (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-dep-join))
		(nth tree 1)
		nil)))
(define planner_tree_ir_dep_join_left (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-dep-join))
		(nth tree 2)
		nil)))
(define planner_tree_ir_dep_join_right (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-dep-join))
		(nth tree 3)
		nil)))
(define planner_tree_ir_dep_join_accessing (lambda (tree)
	(if (and
		(equal? (planner_tree_ir_node_kind tree) (quote op-dep-join))
		(>= (count tree) 5))
		(nth tree 4)
		'())))
(define planner_tree_ir_map_projections (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-map))
		(nth tree 1)
		nil)))
(define planner_tree_ir_groupby_aggs (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-groupby))
		(nth tree 2)
		nil)))
(define planner_tree_ir_groupby_keys (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-groupby))
		(nth tree 1)
		nil)))
(define planner_tree_ir_groupby_having (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-groupby))
		(nth tree 3)
		nil)))
(define planner_tree_ir_groupby_child (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-groupby))
		(nth tree 4)
		nil)))
(define planner_tree_ir_select_predicate (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-select))
		(nth tree 1)
		nil)))
(define planner_tree_ir_select_child (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-select))
		(nth tree 2)
		nil)))
(define planner_tree_ir_join_predicate (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-join))
		(nth tree 2)
		nil)))
(define planner_tree_ir_join_left (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-join))
		(nth tree 3)
		nil)))
(define planner_tree_ir_join_right (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-join))
		(nth tree 4)
		nil)))
(define planner_tree_ir_map_child (lambda (tree)
	(if (equal? (planner_tree_ir_node_kind tree) (quote op-map))
		(nth tree 2)
		nil)))
(define planner_tree_ir_primary_join_node (lambda (tree)
	(match (planner_tree_ir_node_kind tree)
		'op-window (planner_tree_ir_primary_join_node (planner_tree_ir_window_child tree))
		'op-map (planner_tree_ir_primary_join_node (planner_tree_ir_map_child tree))
		'op-groupby (planner_tree_ir_primary_join_node (planner_tree_ir_groupby_child tree))
		'op-select (planner_tree_ir_select_child tree)
		tree)))
(define planner_tree_ir_primary_map_node (lambda (tree)
	(match (planner_tree_ir_node_kind tree)
		'op-window (planner_tree_ir_primary_map_node (planner_tree_ir_window_child tree))
		'op-select (planner_tree_ir_primary_map_node (planner_tree_ir_select_child tree))
		'op-map tree
		nil)))
(define planner_tree_ir_primary_groupby_node (lambda (tree)
	(match (planner_tree_ir_node_kind tree)
		'op-window (planner_tree_ir_primary_groupby_node (planner_tree_ir_window_child tree))
		'op-select (planner_tree_ir_primary_groupby_node (planner_tree_ir_select_child tree))
		'op-groupby tree
		nil)))
(define planner_tree_ir_window_effective_order (lambda (tree fallback_order)
	(match tree
		'((quote op-window) _ order _ _) (if (or (nil? order) (equal? order '()))
			(merge fallback_order '())
			(merge order '()))
		_ (merge fallback_order '()))))
(define planner_tree_ir_window_effective_limit (lambda (tree fallback_limit)
	(match tree
		'((quote op-window) _ _ computations _) (coalesceNil
			(planner_tree_ir_lookup_computation computations (quote limit))
			fallback_limit)
		_ fallback_limit)))
(define planner_tree_ir_window_effective_offset (lambda (tree fallback_offset)
	(match tree
		'((quote op-window) _ _ computations _) (coalesceNil
			(planner_tree_ir_lookup_computation computations (quote offset))
			fallback_offset)
		_ fallback_offset)))
(define planner_tree_ir_window_rewrite_order (lambda (tree fallback_order rewrite_expr)
	(map (planner_tree_ir_window_effective_order tree fallback_order) (lambda (oi) (match oi
		'(col dir) (list (rewrite_expr col) dir)
		oi)))))
(define planner_tree_ir_window_make_group_stage (lambda (tree fallback_order fallback_limit fallback_offset group having rewrite_expr aliases init)
	(make_group_stage
		group
		having
		(planner_tree_ir_window_rewrite_order tree fallback_order rewrite_expr)
		(planner_tree_ir_window_effective_limit tree fallback_limit)
		(planner_tree_ir_window_effective_offset tree fallback_offset)
		aliases
		init)))
(define planner_tree_ir_scan_aliases (lambda (tree)
	(match (planner_tree_ir_node_kind tree)
		'op-scan (begin
			(define table_payload (nth tree 2))
			(match table_payload
				'(alias _ _ _ _) (if (nil? alias) '() (list alias))
				'()))
		'op-join (merge
			(planner_tree_ir_scan_aliases (planner_tree_ir_join_left tree))
			(planner_tree_ir_scan_aliases (planner_tree_ir_join_right tree)))
		'op-dep-join (merge
			(planner_tree_ir_scan_aliases (planner_tree_ir_dep_join_left tree))
			(planner_tree_ir_scan_aliases (planner_tree_ir_dep_join_right tree)))
		'op-select (planner_tree_ir_scan_aliases (planner_tree_ir_select_child tree))
		'op-map (planner_tree_ir_scan_aliases (planner_tree_ir_map_child tree))
		'op-groupby (planner_tree_ir_scan_aliases (planner_tree_ir_groupby_child tree))
		'op-window (planner_tree_ir_scan_aliases (planner_tree_ir_window_child tree))
		'())))
(define planner_tree_ir_expr_refs_aliases (lambda (expr aliases)
	(reduce (extract_tblvars expr) (lambda (found tv)
		(or found (has? aliases tv))) false)))
(define planner_tree_ir_accessing_for_rhs (lambda (tree lhs_aliases)
	(match (planner_tree_ir_node_kind tree)
		'op-scan '()
		'op-select (merge_unique
			(if (planner_tree_ir_expr_refs_aliases (planner_tree_ir_select_predicate tree) lhs_aliases)
				(list (quote select))
				'())
			(planner_tree_ir_accessing_for_rhs (planner_tree_ir_select_child tree) lhs_aliases))
		'op-map (merge_unique
			(if (reduce_assoc (planner_tree_ir_map_projections tree) (lambda (found key v)
				(or found (planner_tree_ir_expr_refs_aliases v lhs_aliases))) false)
				(list (quote map))
				'())
			(planner_tree_ir_accessing_for_rhs (planner_tree_ir_map_child tree) lhs_aliases))
		'op-groupby (merge_unique
			(if (or
				(reduce (coalesceNil (planner_tree_ir_groupby_keys tree) '()) (lambda (found expr)
					(or found (planner_tree_ir_expr_refs_aliases expr lhs_aliases))) false)
				(reduce_assoc (planner_tree_ir_groupby_aggs tree) (lambda (found key v)
					(or found (planner_tree_ir_expr_refs_aliases v lhs_aliases))) false)
				(planner_tree_ir_expr_refs_aliases (planner_tree_ir_groupby_having tree) lhs_aliases))
				(list (quote group))
				'())
			(planner_tree_ir_accessing_for_rhs (planner_tree_ir_groupby_child tree) lhs_aliases))
		'op-window (merge_unique
			(if (or
				(reduce (coalesceNil (nth tree 1) '()) (lambda (found expr)
					(or found (planner_tree_ir_expr_refs_aliases expr lhs_aliases))) false)
				(reduce (coalesceNil (planner_tree_ir_window_order tree) '()) (lambda (found oi) (match oi
					'(col order_dir) (or found (planner_tree_ir_expr_refs_aliases col lhs_aliases))
					_ found)) false)
				(reduce (coalesceNil (nth tree 3) '()) (lambda (found entry) (match entry
					'(_ entry_value) (or found (planner_tree_ir_expr_refs_aliases entry_value lhs_aliases))
					_ found)) false))
				(list (quote window))
				'())
			(planner_tree_ir_accessing_for_rhs (planner_tree_ir_window_child tree) lhs_aliases))
		'op-join (merge_unique
			(if (planner_tree_ir_expr_refs_aliases (planner_tree_ir_join_predicate tree) lhs_aliases)
				(list (quote join))
				'())
			(planner_tree_ir_accessing_for_rhs (planner_tree_ir_join_left tree) lhs_aliases)
			(planner_tree_ir_accessing_for_rhs (planner_tree_ir_join_right tree) lhs_aliases))
		'op-dep-join (merge_unique
			(if (planner_tree_ir_expr_refs_aliases (planner_tree_ir_dep_join_predicate tree) lhs_aliases)
				(list (quote join))
				'())
			(planner_tree_ir_accessing_for_rhs (planner_tree_ir_dep_join_left tree) lhs_aliases)
			(planner_tree_ir_accessing_for_rhs (planner_tree_ir_dep_join_right tree) lhs_aliases))
		'())))
(define planner_tree_ir_collect_accessing_tags (lambda (tree)
	(match (planner_tree_ir_node_kind tree)
		'op-dep-join (merge_unique
			(coalesceNil (planner_tree_ir_dep_join_accessing tree) '())
			(planner_tree_ir_collect_accessing_tags (planner_tree_ir_dep_join_left tree))
			(planner_tree_ir_collect_accessing_tags (planner_tree_ir_dep_join_right tree)))
		'op-join (merge_unique
			(planner_tree_ir_collect_accessing_tags (planner_tree_ir_join_left tree))
			(planner_tree_ir_collect_accessing_tags (planner_tree_ir_join_right tree)))
		'op-select (planner_tree_ir_collect_accessing_tags (planner_tree_ir_select_child tree))
		'op-map (planner_tree_ir_collect_accessing_tags (planner_tree_ir_map_child tree))
		'op-groupby (planner_tree_ir_collect_accessing_tags (planner_tree_ir_groupby_child tree))
		'op-window (planner_tree_ir_collect_accessing_tags (planner_tree_ir_window_child tree))
		'())))
(define annotate_dependent_joins (lambda (tree)
	(match (planner_tree_ir_node_kind tree)
		'op-scan tree
		'op-select (planner_tree_ir_select
			(planner_tree_ir_select_predicate tree)
			(annotate_dependent_joins (planner_tree_ir_select_child tree)))
		'op-map (planner_tree_ir_map
			(planner_tree_ir_map_projections tree)
			(annotate_dependent_joins (planner_tree_ir_map_child tree)))
		'op-groupby (planner_tree_ir_groupby
			(planner_tree_ir_groupby_keys tree)
			(planner_tree_ir_groupby_aggs tree)
			(planner_tree_ir_groupby_having tree)
			(annotate_dependent_joins (planner_tree_ir_groupby_child tree)))
		'op-window (planner_tree_ir_window
			(nth tree 1)
			(planner_tree_ir_window_order tree)
			(nth tree 3)
			(annotate_dependent_joins (planner_tree_ir_window_child tree)))
		'op-join (planner_tree_ir_join
			(nth tree 1)
			(planner_tree_ir_join_predicate tree)
			(annotate_dependent_joins (planner_tree_ir_join_left tree))
			(annotate_dependent_joins (planner_tree_ir_join_right tree)))
		'op-dep-join (begin
			(define annotated_left (annotate_dependent_joins (planner_tree_ir_dep_join_left tree)))
			(define annotated_right (annotate_dependent_joins (planner_tree_ir_dep_join_right tree)))
			(define lhs_aliases (planner_tree_ir_scan_aliases annotated_left))
			(define accessing (planner_tree_ir_accessing_for_rhs annotated_right lhs_aliases))
			(planner_tree_ir_dep_join
				(planner_tree_ir_dep_join_predicate tree)
				annotated_left
				annotated_right
				accessing))
		tree)))
(define planner_tree_ir_select_equivalence_classes (lambda (predicate)
	(filter (map (flatten_and_terms (coalesceNil predicate true)) (lambda (term) (match term
		'((symbol equal??) lhs rhs) (list lhs rhs)
		'((quote equal??) lhs rhs) (list lhs rhs)
		nil)))
		(lambda (entry) (not (nil? entry))))))
(define planner_tree_ir_select_repr_from_cclasses (lambda (cclasses)
	(reduce cclasses (lambda (acc cc) (match cc
		'(lhs rhs) (merge acc
			(list
				(serialize lhs) rhs
				(serialize rhs) lhs))
		_ acc))
		'())))
(define planner_tree_ir_select_repr_lookup (lambda (repr expr)
	(get_assoc repr (serialize expr))))
(define planner_tree_ir_select_rewrite_expr (lambda (expr repr)
	(coalesce
		(planner_tree_ir_select_repr_lookup repr expr)
		(match expr
			(cons sym args) (cons sym
				(map args (lambda (arg)
					(planner_tree_ir_select_rewrite_expr arg repr))))
			expr))))
(define planner_tree_ir_select_rewrite_fields (lambda (fields repr)
	(reduce_assoc fields (lambda (acc k v)
		(merge acc (list k
			(planner_tree_ir_select_rewrite_expr v repr))))
		'())))
(define make_unnest_select_rule_info (lambda (cclasses repr)
	(list cclasses repr)))
(define unnest_select_rule_info_repr (lambda (info)
	(match info
		'(cclass_set repr) repr
		_ '())))
(define unnest_select_rule_apply_expr (lambda (info expr)
	(planner_tree_ir_select_rewrite_expr expr
		(unnest_select_rule_info_repr info))))
(define unnest_select_rule_apply_fields (lambda (info fields)
	(planner_tree_ir_select_rewrite_fields fields
		(unnest_select_rule_info_repr info))))
(define unnest_select_rule_build_info (lambda (select_node) (begin
	(define select_cclasses (if (nil? select_node)
		'()
		(planner_tree_ir_select_equivalence_classes
			(planner_tree_ir_select_predicate select_node))))
	(make_unnest_select_rule_info
		select_cclasses
		(planner_tree_ir_select_repr_from_cclasses select_cclasses)))))
(define unnest_select_rule (lambda (tree)
	(begin
		(define window_node (if (equal? (planner_tree_ir_node_kind tree) (quote op-window)) tree nil))
		(define shape_node (if (nil? window_node) tree (planner_tree_ir_window_child window_node)))
		(define fields_node (if (nil? shape_node) nil
			(if (equal? (planner_tree_ir_node_kind shape_node) (quote op-map))
				(planner_tree_ir_map_projections shape_node)
				(if (equal? (planner_tree_ir_node_kind shape_node) (quote op-groupby))
					(planner_tree_ir_groupby_aggs shape_node)
					nil))))
		(define select_node (if (nil? shape_node) nil
			(if (equal? (planner_tree_ir_node_kind shape_node) (quote op-map))
				(planner_tree_ir_select_child shape_node)
				(if (equal? (planner_tree_ir_node_kind shape_node) (quote op-groupby))
					(nth shape_node 4)
					nil))))
		(define join_tree (if (nil? select_node) nil (planner_tree_ir_select_child select_node)))
		(define select_info (unnest_select_rule_build_info select_node))
		(define rewritten_fields_node
			(unnest_select_rule_apply_fields select_info fields_node))
		(list
			(if (and
				(not (nil? join_tree))
				(equal? (planner_tree_ir_extract_tables join_tree) '())
				(not (reduce_assoc rewritten_fields_node (lambda (a k v) (or a
					(begin
						(define has_nested_aggregate (lambda (e) (match e
							(cons (symbol aggregate) _) true
							(cons s args) (reduce args (lambda (a2 b) (or a2 (has_nested_aggregate b))) false)
							false)))
						(has_nested_aggregate v)))) false)))
				(list (car (extract_assoc rewritten_fields_node (lambda (k v) v))) '())
				nil)
			select_info))))
(define unnest_join_rule (lambda (tree condition_expr inner_aliases outer_ref_rewriter) (begin
	(define join_node (planner_tree_ir_primary_join_node tree))
	(define join_condition (if (equal? (planner_tree_ir_node_kind join_node) (quote op-dep-join))
		(planner_tree_ir_dep_join_predicate join_node)
		condition_expr))
	(define accessing_tags
		(merge_unique
			(if (equal? (planner_tree_ir_node_kind join_node) (quote op-dep-join))
				(planner_tree_ir_dep_join_accessing join_node)
				'())
			(planner_tree_ir_collect_accessing_tags tree)))
	(match (scalar_subselect_correlation_info join_condition inner_aliases outer_ref_rewriter)
		'(us_outer_parts us_domain_cols us_inner_cond_raw)
		(list us_outer_parts us_domain_cols us_inner_cond_raw accessing_tags)))))
(define unnest_window_rule_domain_partitions (lambda (tree us_domain_cols us_accessing_tags)
	(define explicit_partition (coalesceNil (planner_tree_ir_window_partition tree) '()))
	(if (unnest_groupby_rule_requires_domain_keys us_accessing_tags)
		(unnest_window_rule_merge_partitions
			explicit_partition
			(map us_domain_cols (lambda (dc) (nth dc 0))))
		explicit_partition)))
(define unnest_window_rule_bind_domain_partitions (lambda (tree us_domain_cols us_accessing_tags)
	(planner_tree_ir_window_with_partition
		tree
		(unnest_window_rule_domain_partitions tree us_domain_cols us_accessing_tags))))
(define unnest_window_rule_merge_partitions (lambda (base_partition domain_partition)
	(reduce domain_partition (lambda (acc expr)
		(append_unique acc expr))
		(coalesceNil base_partition '()))))
(define unnest_window_rule_rewrite_over (lambda (over domain_partition) (begin
	(define partition_cols (coalesceNil (car over) '()))
	(define order_cols (coalesceNil (cadr over) '()))
	(list
		(unnest_window_rule_merge_partitions partition_cols domain_partition)
		(map order_cols (lambda (oi) (match oi
			'(col dir) (list
				(unnest_window_rule_rewrite_expr col domain_partition)
				dir)
			oi)))))))
(define unnest_window_rule_rewrite_expr (lambda (expr domain_partition)
	(match expr
		'((symbol window_func) fn args over) (list
			(symbol window_func)
			fn
			(map args (lambda (arg)
				(unnest_window_rule_rewrite_expr arg domain_partition)))
			(unnest_window_rule_rewrite_over over domain_partition))
		'((quote window_func) fn args over) (list
			(quote window_func)
			fn
			(map args (lambda (arg)
				(unnest_window_rule_rewrite_expr arg domain_partition)))
			(unnest_window_rule_rewrite_over over domain_partition))
		(cons sym args) (cons sym (map args (lambda (arg)
			(unnest_window_rule_rewrite_expr arg domain_partition))))
		expr)))
(define unnest_window_rule_rewrite_fields (lambda (fields domain_partition)
	(map_assoc fields (lambda (k v)
		(unnest_window_rule_rewrite_expr v domain_partition)))))
(define unnest_window_rule (lambda (tree fields_expr condition_expr us_domain_cols us_accessing_tags us_select_info build_groupby build_map us_has_agg us_has_grp) (begin
	(define domain_partition (unnest_window_rule_domain_partitions tree us_domain_cols us_accessing_tags))
	(define rewritten_fields (unnest_window_rule_rewrite_fields
		(unnest_select_rule_apply_fields us_select_info fields_expr)
		domain_partition))
	(define rewritten_condition (unnest_window_rule_rewrite_expr
		(unnest_select_rule_apply_expr us_select_info condition_expr)
		domain_partition))
	(if (not (equal? (extract_window_funcs (coalesceNil rewritten_condition true)) '()))
		nil
		(begin
			(define rewritten_value_expr (car (extract_assoc rewritten_fields (lambda (k v) v))))
			(if (or us_has_agg us_has_grp)
				(build_groupby rewritten_value_expr)
				(build_map rewritten_value_expr)))))))
(define unnest_accessing_has_tag (lambda (accessing_tags tag)
	(reduce (coalesceNil accessing_tags '()) (lambda (found entry)
		(or found (equal? entry tag))) false)))
(define unnest_groupby_rule_requires_domain_keys (lambda (accessing_tags)
	(or
		(unnest_accessing_has_tag accessing_tags (quote select))
		(unnest_accessing_has_tag accessing_tags (quote map))
		(unnest_accessing_has_tag accessing_tags (quote group))
		(unnest_accessing_has_tag accessing_tags (quote window))
		(unnest_accessing_has_tag accessing_tags (quote join)))))
(define unnest_groupby_rule_is_static_group (lambda (orig_group)
	(or
		(nil? orig_group)
		(equal? orig_group '())
		(equal? orig_group '(1)))))
(define unnest_groupby_rule_static_group_keys (lambda (dom_group_cols accessing_tags)
	(if (not (equal? dom_group_cols '()))
		dom_group_cols
		'(1))))
(define unnest_groupby_rule_build_group_keys (lambda (orig_group dom_group_cols accessing_tags static_group)
	(if static_group
		(unnest_groupby_rule_static_group_keys dom_group_cols accessing_tags)
		(if (not (equal? dom_group_cols '()))
			(merge dom_group_cols orig_group)
			(if (unnest_groupby_rule_requires_domain_keys accessing_tags)
				(merge dom_group_cols orig_group)
				orig_group)))))
(define exists_subquery_uses_session_state_for_row_existence (lambda (query)
	(match query
		'(schema2 tables2 fields_with_window condition2 group2 having2 order_with_window limit2 offset2)
		(expr_uses_session_state
			(list schema2 tables2 '() condition2 group2 having2 nil limit2 offset2))
		(expr_uses_session_state query))))
(define count_subquery_cache_policy (lambda (query target_expr)
	(match query
		'(s t f c g h o l off) (begin
			(define only_count (match f
				'("__cnt" ((quote aggregate) 1 op 0)) (equal?? op (quote +))
				'("__cnt" ((symbol aggregate) 1 op 0)) (equal?? op (quote +))
				false))
			(define session_sensitive_count
				(if (nil? target_expr)
					(exists_subquery_uses_session_state_for_row_existence query)
					(expr_uses_session_state query)))
			(if (and only_count (equal? g '(1)) session_sensitive_count)
				(quote uncached-count)
				nil))
		nil)))
(define unnest_map_rule_projection_expr (lambda (map_node field_name)
	(if (nil? map_node)
		nil
		(reduce_assoc (planner_tree_ir_map_projections map_node) (lambda (found proj_name proj_expr)
			(if (or (not (nil? found)) (not (equal?? proj_name field_name)))
				found
				proj_expr))
			nil))))
(define unnest_map_rule_rewrite_expr (lambda (map_node expr)
	(match expr
		'((symbol get_column) alias_ ti col ci) (if (nil? alias_)
			(coalesceNil (unnest_map_rule_projection_expr map_node col) expr)
			expr)
		'((quote get_column) alias_ ti col ci) (if (nil? alias_)
			(coalesceNil (unnest_map_rule_projection_expr map_node col) expr)
			expr)
		(cons sym args) (cons (unnest_map_rule_rewrite_expr map_node sym) (map args (lambda (arg)
			(unnest_map_rule_rewrite_expr map_node arg))))
		expr)))
(define unnest_map_rule (lambda (tree subquery sq_cache target_expr us_single_tbl us_nested_direct_tbls us_base_aliases us_base_tables us_has_stages us_own_stages us_inner_aliases tables2_us us_has_outer us_inner_stages us_domain_cols us_ria us_sq_prefix us_lookup us_alias_map us_outer_parts us_ror us_inner_cond_raw schemas2_us us_value_expr us_accessing_tags us_select_info) (begin
	(define map_node (planner_tree_ir_primary_map_node tree))
	(define us_rewrite_map_expr (lambda (expr)
		(unnest_map_rule_rewrite_expr map_node
			(unnest_select_rule_apply_expr us_select_info expr))))
	(define us_nested_direct_refs_base_aliases (reduce us_nested_direct_tbls (lambda (acc td) (match td
		'(_ _ _ _ je) (or acc
			(and (not (nil? je))
				(reduce (extract_tblvars je) (lambda (found tv)
					(or found (has? us_base_aliases tv))) false)))
		_ acc))
		false))
	(if (and us_single_tbl (not us_nested_direct_refs_base_aliases))
		(begin
			(define us_tdesc (car us_base_tables))
			(define us_tblvar (nth us_tdesc 0))
			(define us_tbl_schema (nth us_tdesc 1))
			(define us_tbl_name (nth us_tdesc 2))
			(define us_stage_order_fallback (if us_has_stages (coalesceNil (stage_order_list (car us_own_stages)) '()) '()))
			(define us_stage_limit_fallback (if us_has_stages (stage_limit_val (car us_own_stages)) nil))
			(define us_stage_offset_fallback (if us_has_stages (stage_offset_val (car us_own_stages)) nil))
			(define us_orig_order (planner_tree_ir_window_effective_order tree us_stage_order_fallback))
			(define us_orig_limit (planner_tree_ir_window_effective_limit tree us_stage_limit_fallback))
			(define us_orig_offset (planner_tree_ir_window_effective_offset tree us_stage_offset_fallback))
			(define us_inner_tbls (filter tables2_us (lambda (t) (match t '(a _ _ _ _) (has? us_inner_aliases a) false))))
			(define us_rewrite_table_expr (lambda (expr)
				(us_ria (us_ror expr))))
			(define us_inner_tbls_rewritten (scalar_subselect_rewrite_tables us_inner_tbls us_rewrite_table_expr))
			(define us_simple_uncorrelated_cache_key (if (and
				(not us_has_outer)
				(equal? us_inner_tbls '())
				(equal? us_inner_stages '()))
				(serialize subquery)
				nil))
			(define us_cached_subst (if (nil? us_simple_uncorrelated_cache_key)
				nil
				(get_assoc (coalesceNil (sq_cache "scalar_helper_cache") '()) us_simple_uncorrelated_cache_key)))
			(if (not (nil? us_cached_subst))
				(list us_cached_subst '())
				(begin
					(if (not (equal? us_inner_tbls_rewritten '()))
						(sq_cache "tables" (merge us_inner_tbls_rewritten (coalesceNil (sq_cache "tables") '()))))
					(define us_partition_exprs
						(merge
							(coalesce
								(planner_tree_ir_window_partition tree)
								(map us_domain_cols (lambda (dc) (nth dc 0)))
								'())
							'()))
								(define us_renamed_order
									(scalar_scan_bind_unqualified_order_alias
										(scalar_scan_rewrite_order us_orig_order us_ria)
										us_sq_prefix))
					(define us_order_supported (scalar_scan_order_supported us_renamed_order us_sq_prefix))
					(if (not us_order_supported)
						nil
						(begin
							(define us_outer_sources (domain_outer_sources_from_correlation_cols us_domain_cols us_ria))
							(define us_inner_stages_rewritten (scalar_subselect_rewrite_stages_with_lookup
								us_inner_stages
								us_ria
								us_lookup))
							(define us_nested_outer_sources (scalar_subselect_collect_stage_outer_sources us_inner_stages_rewritten))
							(define us_part_stage (planner_tree_ir_window_make_scalar_partition_stage
								tree
								us_partition_exprs
								us_ria
								us_sq_prefix
								us_orig_order
								us_orig_limit
								us_orig_offset
								(list us_sq_prefix)
								(merge_unique (list us_outer_sources us_nested_outer_sources))))
							(sq_cache "groups" (merge
								(list us_part_stage)
								us_inner_stages_rewritten
								(coalesceNil (sq_cache "groups") '())))
							(define us_join_lim (map us_outer_parts (lambda (p) (us_ria (us_ror p)))))
							(define us_inner_lim (us_ria us_inner_cond_raw))
							(define us_full_lim (if (nil? us_inner_lim)
								(if (equal? (count us_join_lim) 0) true (if (equal? (count us_join_lim) 1) (car us_join_lim) (cons (quote and) us_join_lim)))
								(cons (quote and) (merge us_join_lim (list us_inner_lim)))))
							(define us_nested_direct_tbls_rewritten (scalar_subselect_rewrite_tables us_nested_direct_tbls us_rewrite_table_expr))
							(define us_tbl_entries (merge
								(list (list us_sq_prefix us_tbl_schema (make_unnest_helper_table us_tbl_schema us_tbl_name (quote scalar)) true us_full_lim))
								us_nested_direct_tbls_rewritten))
							(define us_inner_schema (schemas2_us us_tblvar))
							(define us_passthrough_schemas (merge
								(if (not (nil? us_inner_schema)) (list us_sq_prefix us_inner_schema) '())
								(scalar_subselect_passthrough_schemas (merge us_inner_tbls us_nested_direct_tbls) schemas2_us)))
							(if (not (equal? us_passthrough_schemas '()))
								(sq_cache "schemas" (merge us_passthrough_schemas (coalesceNil (sq_cache "schemas") '()))))
							(define us_presence_col (if (nil? us_inner_schema)
								nil
								(coalesce
									(reduce us_inner_schema (lambda (found coldef)
										(if (not (nil? found)) found
											(if (equal? (coldef "Key") "PRI") (coldef "Field") nil)))
										nil)
									(match us_inner_schema
										(cons first_col _) (first_col "Field")
										nil))))
							(define us_subst_raw (us_ria (us_rewrite_map_expr us_value_expr)))
							(define us_subst (if (nil? us_presence_col)
								us_subst_raw
								(list (quote if)
									(list (quote nil?) (list (quote get_column) us_sq_prefix false us_presence_col false))
									nil
									us_subst_raw)))
							(if (not (nil? us_simple_uncorrelated_cache_key))
								(sq_cache "scalar_helper_cache"
									(set_assoc (coalesceNil (sq_cache "scalar_helper_cache") '())
										us_simple_uncorrelated_cache_key
										us_subst)))
							(list us_subst us_tbl_entries))))))
		/* === Multi-table scalar subselect (BTW2025 §3.2 + cclass elim per FAQ §40) ===
		Inner has 2+ base tables (joined via ON or comma). After simpleDJoinElimination
		(pulling outer-correlation σ up into the dep-join), the right side is a regular
		multi-table join. We emit the inner tables prefix-renamed into the outer's
		tables list, with the first base table carrying isOuter=true and the combined
		outer-correlation + inner-condition as its join expression. The value
		expression substitutes via us_ria (renames inner aliases to prefixed names).
		Limit/ORDER per outer key are not handled in this branch — those still fall
		through to nil and need the deferred-mat or partition-stage path. */
		(if us_nested_direct_refs_base_aliases
			nil /* nested direct table refs base aliases: not handled */
			(begin
				(define us_stage_order_fallback (if us_has_stages (coalesceNil (stage_order_list (car us_own_stages)) '()) '()))
				(define us_stage_limit_fallback (if us_has_stages (stage_limit_val (car us_own_stages)) nil))
				(define us_stage_offset_fallback (if us_has_stages (stage_offset_val (car us_own_stages)) nil))
				(define us_orig_order (planner_tree_ir_window_effective_order tree us_stage_order_fallback))
				(define us_orig_limit (planner_tree_ir_window_effective_limit tree us_stage_limit_fallback))
				(define us_orig_offset (planner_tree_ir_window_effective_offset tree us_stage_offset_fallback))
				/* FAQ §43: ORDER BY / LIMIT / OFFSET per outer binding is realised by
				planner_tree_ir_window_make_scalar_partition_stage below, which already
				receives us_orig_order / us_orig_limit / us_orig_offset. The previous
				nil-bailout for the multi-table case was a conservative guard from before
				the partition-stage path covered multi-table inputs. */
				(begin
						(define us_inner_tbls (filter tables2_us (lambda (t) (match t '(a _ _ _ _) (has? us_inner_aliases a) false))))
						(define us_rewrite_table_expr (lambda (expr)
							(us_ria (us_ror expr))))
						(define us_inner_tbls_rewritten (scalar_subselect_rewrite_tables us_inner_tbls us_rewrite_table_expr))
						(if (not (equal? us_inner_tbls_rewritten '()))
							(sq_cache "tables" (merge us_inner_tbls_rewritten (coalesceNil (sq_cache "tables") '()))))
						(define us_partition_exprs
							(merge
								(coalesce
									(planner_tree_ir_window_partition tree)
									(map us_domain_cols (lambda (dc) (nth dc 0)))
									'())
								'()))
						(define us_outer_sources (domain_outer_sources_from_correlation_cols us_domain_cols us_ria))
						(define us_inner_stages_rewritten (scalar_subselect_rewrite_stages_with_lookup
							us_inner_stages
							us_ria
							us_lookup))
						(define us_nested_outer_sources (scalar_subselect_collect_stage_outer_sources us_inner_stages_rewritten))
						/* Partition stage keyed on every prefixed inner alias enforces
						scalar (once_limit=2) semantics across the multi-table join. */
						(define us_prefixed_aliases (map us_base_aliases (lambda (a)
							(coalesceNil (us_lookup a) a))))
						(define us_part_stage (planner_tree_ir_window_make_scalar_partition_stage
							tree
							us_partition_exprs
							us_ria
							us_sq_prefix
							us_orig_order
							us_orig_limit
							us_orig_offset
							us_prefixed_aliases
							(merge_unique (list us_outer_sources us_nested_outer_sources))))
						(sq_cache "groups" (merge
							(list us_part_stage)
							us_inner_stages_rewritten
							(coalesceNil (sq_cache "groups") '())))
						(define us_join_lim (map us_outer_parts (lambda (p) (us_ria (us_ror p)))))
						(define us_inner_lim (us_ria us_inner_cond_raw))
						(define us_full_lim (if (nil? us_inner_lim)
							(if (equal? (count us_join_lim) 0) true (if (equal? (count us_join_lim) 1) (car us_join_lim) (cons (quote and) us_join_lim)))
							(cons (quote and) (merge us_join_lim (list us_inner_lim)))))
						(define us_nested_direct_tbls_rewritten (scalar_subselect_rewrite_tables us_nested_direct_tbls us_rewrite_table_expr))
						/* Prefix-wrap all base tables. The first base table carries
						isOuter=true plus the combined correlation+inner-condition
						(us_full_lim). The remaining base tables keep their original
						join expression, rewritten for the new prefixed aliases. */
						(define us_prefixed_base_tables (scalar_subselect_prefixed_tables us_base_tables us_lookup us_rewrite_table_expr))
						(define us_first_base (car us_prefixed_base_tables))
						(define us_rest_base (cdr us_prefixed_base_tables))
						(define us_first_patched (match us_first_base
							'(a s t _io _je) (list a s t true us_full_lim)
							us_first_base))
						(define us_tbl_entries (merge
							(list us_first_patched)
							us_rest_base
							us_nested_direct_tbls_rewritten))
						/* Passthrough schemas for the prefix-renamed base aliases. */
						(define us_passthrough_schemas (scalar_subselect_prefixed_schemas us_prefixed_base_tables us_alias_map schemas2_us))
						(if (not (equal? us_passthrough_schemas '()))
							(sq_cache "schemas" (merge us_passthrough_schemas (coalesceNil (sq_cache "schemas") '()))))
						(define us_subst (us_ria (us_rewrite_map_expr us_value_expr)))
						(list us_subst us_tbl_entries)))))))))
(define unnest_groupby_rule (lambda (tree subquery sq_cache target_expr tables2_us us_lookup us_alias_map us_ria us_has_stages us_own_stages us_inner_stages us_domain_cols us_inner_cond_raw schemas2_us us_value_expr us_has_grp us_accessing_tags us_select_info) (begin
	/* === A: Aggregate -> flatten inner tables + scoped GROUP stage ===
	Neumann Γ_{A∪D;f}: add domain cols to GROUP BY, flatten inner tables
	with prefix into outer table list. No materialization. */
	(define us_prefix_ria (lambda (expr)
		(scalar_subselect_rewrite_prefixed_expr
			(unnest_select_rule_apply_expr us_select_info expr)
			us_lookup)))
	(define us_prefix_table_expr (lambda (expr)
		(us_prefix_ria (unnest_runtime_outer_ref_expr expr))))
	(define us_prefixed_tables (scalar_subselect_prefixed_tables tables2_us us_lookup us_prefix_table_expr))
	(define us_local_aliases (scalar_subselect_table_aliases tables2_us))
	(define us_inner_cond_prefixed (if (nil? us_inner_cond_raw) nil (us_prefix_ria us_inner_cond_raw)))
	(define us_groupby_node (planner_tree_ir_primary_groupby_node tree))
	(define us_stage_group_fallback (if us_has_stages (coalesceNil (stage_group_cols (car us_own_stages)) '()) '()))
	(define us_stage_having_fallback (if us_has_stages (stage_having_expr (car us_own_stages)) nil))
	(define us_orig_group (if (nil? us_groupby_node)
		us_stage_group_fallback
		(coalesceNil (planner_tree_ir_groupby_keys us_groupby_node) '())))
	(define us_orig_having (if (nil? us_groupby_node)
		us_stage_having_fallback
		(planner_tree_ir_groupby_having us_groupby_node)))
	(define us_cache_policy (count_subquery_cache_policy subquery target_expr))
	(define us_rewrite_domain_outer_expr (lambda (expr)
		(us_prefix_ria (unnest_runtime_outer_ref_expr expr))))
	(define us_nested_domain_cols (reduce us_inner_stages (lambda (acc s)
		(merge acc
			(filter (map (coalesceNil (stage_outer_sources s) '()) (lambda (src)
				(match src
					'(outer_tv outer_col inner_expr)
					(if (reduce us_local_aliases (lambda (found local_alias)
						(or found (equal?? local_alias outer_tv))) false)
						nil
						(list inner_expr
							(us_rewrite_domain_outer_expr
								(list (quote get_column) outer_tv false outer_col false))))
					_ nil)))
				(lambda (x) (not (nil? x)))))) '()))
	(define us_domain_cols_all (reduce (merge us_domain_cols us_nested_domain_cols) (lambda (acc dc)
		(if (reduce acc (lambda (found existing) (or found (equal? existing dc))) false)
			acc
			(merge acc (list dc)))) '()))
	(define us_dom_group_cols (map us_domain_cols_all (lambda (dc) (us_prefix_ria (nth dc 0)))))
	(define us_prefixed_aliases (scalar_subselect_table_aliases us_prefixed_tables))
	(define us_static_group (and
		(not us_has_grp)
		(unnest_groupby_rule_is_static_group us_orig_group)))
	(define us_new_group (unnest_groupby_rule_build_group_keys
		(map us_orig_group us_prefix_ria)
		us_dom_group_cols
		us_accessing_tags
		us_static_group))
	(define us_new_having (if (nil? us_orig_having) nil (us_prefix_ria us_orig_having)))
	(define us_stage_aliases (if (equal? us_prefixed_aliases '()) nil us_prefixed_aliases))
	(define us_stage_order_fallback_a (if (and us_has_grp us_has_stages) (coalesceNil (stage_order_list (car us_own_stages)) '()) '()))
	(define us_stage_limit_fallback_a (if (and us_has_grp us_has_stages) (stage_limit_val (car us_own_stages)) nil))
	(define us_stage_offset_fallback_a (if (and us_has_grp us_has_stages) (stage_offset_val (car us_own_stages)) nil))
	/* FAQ static-group rule: when the inner subquery has no explicit GROUP BY
	but the value is an aggregate (NK15 §3.2 Γ_{A∪D;f}), the aggregate produces
	exactly one row per domain binding. Inner ORDER/LIMIT/OFFSET on top of a
	single row are no-ops (LIMIT≥1, OFFSET=0/nil) and must not leak into the
	scoped GROUP stage's stage_limit — that would erroneously cap the outer
	stream at one global row. The raw subquery shape still carries
	`(op-window nil order (limit offset) shaped_tree)`, so explicitly drop the
	window-level limit/offset/order for the static-group path. */
	(define us_static_group_drop_window_limit (and us_static_group
		(or (nil? us_stage_limit_fallback_a) (>= us_stage_limit_fallback_a 1))
		(or (nil? us_stage_offset_fallback_a) (equal? us_stage_offset_fallback_a 0))))
	(define us_new_order (if us_static_group_drop_window_limit
		'()
		(planner_tree_ir_window_rewrite_order tree us_stage_order_fallback_a us_prefix_ria)))
	(define us_new_limit (if us_static_group_drop_window_limit
		us_stage_limit_fallback_a
		(planner_tree_ir_window_effective_limit tree us_stage_limit_fallback_a)))
	(define us_new_offset (if us_static_group_drop_window_limit
		us_stage_offset_fallback_a
		(planner_tree_ir_window_effective_offset tree us_stage_offset_fallback_a)))
	(define us_group_stage (if (group_stage_requested us_new_group us_new_having us_new_order us_new_limit us_new_offset)
		(stage_with_cache_query
			(stage_with_cache_policy
				(make_group_stage
					us_new_group
					us_new_having
					us_new_order
					us_new_limit
					us_new_offset
					us_stage_aliases
					nil)
				us_cache_policy)
			(if (nil? us_cache_policy) nil subquery))
		nil))
	(define us_prefixed_inner_stages (scalar_subselect_rewrite_stages_with_lookup
		us_inner_stages
		us_prefix_ria
		us_lookup))
	(sq_cache "tables" (merge us_prefixed_tables (coalesceNil (sq_cache "tables") '())))
	(sq_cache "groups" (merge (if (nil? us_group_stage) '() (list us_group_stage)) us_prefixed_inner_stages (coalesceNil (sq_cache "groups") '())))
	(define us_prefixed_schemas (scalar_subselect_prefixed_schemas us_prefixed_tables us_alias_map schemas2_us))
	(sq_cache "schemas" (merge us_prefixed_schemas (coalesceNil (sq_cache "schemas") '())))
	(define us_dom_je_parts (map us_domain_cols_all (lambda (dc)
		(list (quote equal??)
			(us_prefix_ria (nth dc 0))
			(us_rewrite_domain_outer_expr (nth dc 1))))))
	(define us_dom_je (if (equal? (count us_dom_je_parts) 0) true
		(if (equal? (count us_dom_je_parts) 1) (car us_dom_je_parts)
			(cons (quote and) us_dom_je_parts))))
	(define us_inner_parts_list (if (nil? us_inner_cond_prefixed) '()
		(match us_inner_cond_prefixed
			(cons (symbol and) parts) parts
			(cons (quote and) parts) parts
			(list us_inner_cond_prefixed))))
	(define us_expr_refs (lambda (expr) (match expr
		'((symbol get_column) tv _ _ _) (if (nil? tv) '() (list tv))
		'((quote get_column) tv _ _ _) (if (nil? tv) '() (list tv))
		(cons _ args) (reduce args (lambda (acc a) (merge acc (us_expr_refs a))) '())
		'())))
	(define us_last_alias (lambda (part) (begin
		(define part_refs (us_expr_refs part))
		(reduce us_prefixed_aliases (lambda (best al)
			(if (reduce part_refs (lambda (found r) (or found (equal?? r al))) false)
				al best)) nil))))
	(define us_parts_for (lambda (alias) (begin
		(define my_part (filter us_inner_parts_list (lambda (p) (equal?? (us_last_alias p) alias))))
		(if (equal? (count my_part) 0) nil
			(if (equal? (count my_part) 1) (car my_part)
				(cons (quote and) my_part))))))
	(define us_merge_unique_and (lambda (expr_a expr_b)
		(combine_and_terms
			(reduce
				(merge
					(flatten_and_terms expr_a)
					(flatten_and_terms expr_b))
				(lambda (acc part) (append_unique acc part))
				'()))))
	(if (not (nil? us_prefixed_tables))
		(sq_cache "tables" (begin
			(define all_tbls (sq_cache "tables"))
			(define first_alias (match (car us_prefixed_tables) '(a _ _ _ _) a ""))
			(map all_tbls (lambda (td) (match td
				'(a s t io je) (if (not (reduce us_prefixed_aliases (lambda (f al) (or f (equal?? al a))) false)) td
					(begin
						(define my_cond_part (us_parts_for a))
						(if (equal? a first_alias)
							(list a s t true
								(if (nil? my_cond_part) us_dom_je
									(if (equal? us_dom_je true) my_cond_part
										(us_merge_unique_and us_dom_je my_cond_part))))
							(list a s t io
								(if (nil? my_cond_part) je
									(if (nil? je) my_cond_part
										(us_merge_unique_and je my_cond_part)))))))
				td))))))
	(define us_subst_raw (us_prefix_ria
		(unnest_select_rule_apply_expr us_select_info us_value_expr)))
	(define us_is_count (match us_value_expr
		'((symbol aggregate) _ (symbol +) 0) true
		'((quote aggregate) _ (symbol +) 0) true
		'((quote aggregate) _ '(symbol +) 0) true
		false))
	(define us_subst (if us_is_count (list (quote coalesceNil) us_subst_raw 0) us_subst_raw))
	(list us_subst '()))))
(define planner_flat_tables_to_tree_ir (lambda (schema tables)
	(if (or (nil? tables) (equal? tables '()))
		(planner_tree_ir_scan schema nil)
		(reduce (cdr tables) (lambda (left td)
			(planner_tree_ir_join
				(match td
					'(_ _ _ outer_flag _) (if outer_flag (quote left) (quote inner))
					(quote inner))
				(match td
					'(_ _ _ _ joinexpr) joinexpr
					nil)
				left
				(planner_tree_ir_scan schema td)))
			(planner_tree_ir_scan schema (car tables))))))
(define planner_tree_ir_extract_schema (lambda (node) (begin
	(define node_kind (planner_tree_ir_node_kind node))
	(if (equal? node_kind (quote op-scan))
		(nth node 1)
		(if (or
			(equal? node_kind (quote op-join))
			(equal? node_kind (quote op-dep-join)))
			(planner_tree_ir_extract_schema (nth node 3))
			(error (concat "TREE_IR_SCHEMA_EXPECTED_SCAN " (serialize node))))))))
(define planner_tree_ir_extract_tables (lambda (node) (begin
	(define node_kind (planner_tree_ir_node_kind node))
	(if (equal? node_kind (quote op-scan))
		(begin
			(define table_payload (nth node 2))
			(if (nil? table_payload) '() (list table_payload)))
		(if (or
			(equal? node_kind (quote op-join))
			(equal? node_kind (quote op-dep-join)))
			(merge
				(planner_tree_ir_extract_tables (nth node 3))
				(planner_tree_ir_extract_tables (nth node 4)))
			(error (concat "TREE_IR_TABLES_EXPECTED_JOIN " (serialize node))))))))
(define planner_flat_subquery_to_tree_ir (lambda (subquery) (match subquery
	'(schema tables fields condition group having order limit offset) (begin
		(define join_tree (planner_flat_tables_to_tree_ir schema tables))
		(define selected_tree (planner_tree_ir_select condition join_tree))
		(define shaped_tree (if (or
			(not (or (nil? group) (equal? group '())))
			(not (nil? having)))
			(planner_tree_ir_groupby group fields having selected_tree)
			(planner_tree_ir_map fields selected_tree)))
		(planner_tree_ir_window nil order
			(planner_tree_ir_window_computations limit offset)
			shaped_tree))
	_ (error (concat "TREE_IR_NON_FLAT_SUBQUERY " (serialize subquery))))))
(define planner_flat_subquery_roundtrip_via_tree_ir (lambda (subquery)
	(planner_tree_ir_to_flat_subquery
		(planner_flat_subquery_to_tree_ir subquery))))
(define planner_tree_ir_to_flat_subquery (lambda (tree) (begin
	(define tree_kind (planner_tree_ir_node_kind tree))
	(if (not (equal? tree_kind (quote op-window)))
		(error (concat "TREE_IR_EXPECTED_WINDOW_ROOT " (serialize tree)))
		nil)
	(define partition (nth tree 1))
	(define order (nth tree 2))
	(define computations (nth tree 3))
	(define child (nth tree 4))
	(define child_kind (planner_tree_ir_node_kind child))
	(if (not (nil? partition))
		(error (concat "TREE_IR_WINDOW_PARTITION_UNSUPPORTED " (serialize tree)))
		nil)
	(if (equal? child_kind (quote op-map))
		(begin
			(define fields (nth child 1))
			(define select_node (nth child 2))
			(if (not (equal? (planner_tree_ir_node_kind select_node) (quote op-select)))
				(error (concat "TREE_IR_EXPECTED_SELECT_UNDER_MAP " (serialize tree)))
				nil)
			(define condition (nth select_node 1))
			(define join_tree (nth select_node 2))
			(list
				(planner_tree_ir_extract_schema join_tree)
				(planner_tree_ir_extract_tables join_tree)
				fields
				condition
				nil
				nil
				order
				(planner_tree_ir_lookup_computation computations (quote limit))
				(planner_tree_ir_lookup_computation computations (quote offset))))
		(if (equal? child_kind (quote op-groupby))
			(begin
				(define group (nth child 1))
				(define fields (nth child 2))
				(define having (nth child 3))
				(define select_node (nth child 4))
				(if (not (equal? (planner_tree_ir_node_kind select_node) (quote op-select)))
					(error (concat "TREE_IR_EXPECTED_SELECT_UNDER_GROUPBY " (serialize tree)))
					nil)
				(define condition (nth select_node 1))
				(define join_tree (nth select_node 2))
				(list
					(planner_tree_ir_extract_schema join_tree)
					(planner_tree_ir_extract_tables join_tree)
					fields
					condition
					group
					having
					order
					(planner_tree_ir_lookup_computation computations (quote limit))
					(planner_tree_ir_lookup_computation computations (quote offset))))
			(error (concat "TREE_IR_EXPECTED_MAP_OR_GROUPBY " (serialize tree))))))))

(define planner_tree_ir_window_partition_order (lambda (tree fallback_partition_exprs rewrite_inner_expr sq_alias)
	(scalar_scan_partition_order
		(merge (coalesce (planner_tree_ir_window_partition tree) fallback_partition_exprs '()) '())
		rewrite_inner_expr
		sq_alias)))
(define planner_tree_ir_window_partition_count (lambda (tree fallback_partition_exprs)
	(count (merge (coalesce (planner_tree_ir_window_partition tree) fallback_partition_exprs '()) '()))))
(define planner_tree_ir_window_make_scalar_partition_stage (lambda (tree fallback_partition_exprs rewrite_inner_expr sq_alias order_list limit_value offset_value aliases outer_sources)
	(make_scalar_partition_stage
		(merge
			(planner_tree_ir_window_partition_order tree fallback_partition_exprs rewrite_inner_expr sq_alias)
			(scalar_scan_bind_unqualified_order_alias
				(scalar_scan_rewrite_order order_list rewrite_inner_expr)
				sq_alias))
		limit_value
		offset_value
		(planner_tree_ir_window_partition_count tree fallback_partition_exprs)
		aliases
		outer_sources)))
