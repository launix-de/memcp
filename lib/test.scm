/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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

(begin /* own enclosure */
	(print "performing unit tests ...")

	(set teststat (newsession))
	(teststat "count" 0)
	(teststat "success" 0)
	(define assert (lambda (val1 val2 errormsg) (begin
		(teststat "count" (+ (teststat "count") 1))
		(if (equal? val1 val2) (teststat "success" (+ (teststat "success") 1)) (print "failed test "(teststat "count")": " errormsg))
	)))

	/* equal? */
	(assert (equal? "a" "a") true "equality check")
	(assert (equal? "a" "b") false "inequality check")
	/* equal? cross-type coverage (compare.go:Equal) */
	(assert (equal? nil nil) true "equal? nil nil")
	(assert (equal? nil 0) true "equal? nil vs 0 (falsy)")
	(assert (equal? nil 1) false "equal? nil vs 1 (truthy)")
	(assert (equal? 0 nil) true "equal? 0 vs nil (falsy)")
	(assert (equal? 1 nil) false "equal? 1 vs nil (truthy)")
	(assert (equal? nil false) true "equal? nil vs false")
	(assert (equal? nil "") true "equal? nil vs empty string")
	(assert (equal? true true) true "equal? bool same true")
	(assert (equal? false false) true "equal? bool same false")
	(assert (equal? true false) false "equal? bool different")
	(assert (equal? 3.14 3.14) true "equal? float same")
	(assert (equal? 3.14 2.71) false "equal? float different")
	(assert (equal? 3 3.0) true "equal? int vs float")
	(assert (equal? 3.0 3) true "equal? float vs int")
	(assert (equal? 42 "42") true "equal? int vs string")
	(assert (equal? "42" 42) true "equal? string vs int")
	(assert (equal? 3.14 "3.14") true "equal? float vs string")
	(assert (equal? "3.14" 3.14) true "equal? string vs float")
	(assert (equal? 1 true) true "equal? int 1 vs true")
	(assert (equal? 0 false) true "equal? int 0 vs false")
	(assert (equal? 1.0 true) true "equal? float 1.0 vs true")
	(assert (equal? "true" true) true "equal? string vs bool true")
	(assert (equal? '(1 2) '(1 2 3)) false "equal? unequal length lists")
	(assert (equal? '(1 2) '(1 3)) false "equal? lists different elements")
	(assert (equal? '() false) true "equal? empty list vs false")
	(assert (equal? + +) true "equal? same native function")
	/* equal? additional cross-type branches (compare.go:Equal) */
	(assert (equal? true (count '(1))) true "equal? bool vs tagInt (truthy)")
	(assert (equal? false (count '(1))) false "equal? bool false vs tagInt 1")
	(assert (equal? (parse_date "2024-06-15") "2024-06-15") true "equal? date vs string match")
	(assert (equal? (parse_date "2024-06-15") "2025-01-01") false "equal? date vs string mismatch")
	(assert (equal? (parse_date "2024-06-15") (strlen "hello")) false "equal? date vs tagInt fallback")
	(assert (equal? "2024-06-15" (parse_date "2024-06-15")) true "equal? string vs date match")
	(assert (equal? "invalid" (parse_date "2024-06-15")) false "equal? string vs date parse fail")
	(assert (equal? "5" (strlen "hello")) true "equal? string vs tagInt match")
	(assert (equal? "99" (strlen "hello")) false "equal? string vs tagInt mismatch")
	(define fd_eq1 (reduce (produceN 20) (lambda (acc i) (set_assoc acc (concat "k" i) i)) '()))
	(define fd_eq2 (reduce (produceN 20) (lambda (acc i) (set_assoc acc (concat "k" i) i)) '()))
	(assert (equal? fd_eq1 fd_eq2) true "equal? FastDict same content")
	/* equal?? (EqualSQL) cross-type coverage (compare.go:EqualSQL) */
	(assert (nil? (equal?? nil nil)) true "equal?? nil nil returns nil")
	(assert (nil? (equal?? nil 42)) true "equal?? nil int returns nil")
	(assert (nil? (equal?? 42 nil)) true "equal?? int nil returns nil")
	(assert (equal?? 1 1.0) true "equal?? int vs float")
	(assert (equal?? 1.0 1) true "equal?? float vs int")
	(assert (equal?? "42" 42) true "equal?? string vs int")
	(assert (equal?? 42 "42") true "equal?? int vs string")
	(assert (equal?? 3.14 "3.14") true "equal?? float vs string")
	(assert (equal?? "3.14" 3.14) true "equal?? string vs float")
	(assert (equal?? "true" true) true "equal?? string vs bool")
	(assert (equal?? true true) true "equal?? bool same")
	(assert (equal?? + +) true "equal?? same func")
	(assert (equal?? + -) false "equal?? different func")
	/* equal?? additional cross-type branches (compare.go:EqualSQL) */
	(assert (equal?? (parse_date "2024-01-01") (parse_date "2024-01-01")) true "equal?? date same")
	(assert (equal?? (parse_date "2024-01-01") (parse_date "2025-01-01")) false "equal?? date different")
	(assert (equal?? '(1 2) '(1 2)) true "equal?? list same")
	(assert (equal?? '(1 2) '(1 3)) false "equal?? list different")
	(assert (equal?? (parse_date "2024-06-15") "2024-06-15") true "equal?? date vs string")
	(assert (equal?? (parse_date "2024-06-15") (strlen "hello")) false "equal?? date vs tagInt fallback")
	(assert (equal?? 3.0 (count '(1 2 3))) true "equal?? float vs tagInt match")
	(assert (equal?? 3.14 (parse_date "2024-06-15")) false "equal?? float vs date fallback")
	(assert (equal?? "2024-06-15" (parse_date "2024-06-15")) true "equal?? string vs date")
	(assert (equal?? "3" (count '(1 2 3))) true "equal?? string vs tagInt")
	(assert (equal?? true (count '(1))) true "equal?? bool cross-type truthy")
	(assert (equal?? false (count '(1))) false "equal?? bool cross-type falsy")
	(assert (equal?? '() false) true "equal?? empty list vs false")
	(assert (equal?? + (count '(1))) false "equal?? func vs non-func")
	/* Public comparisons preserve SQL UNKNOWN. Internal storage ordering calls
	Less directly and retains a total order including nil. */
	(assert (nil? (< nil nil)) true "less nil nil is unknown")
	(assert (nil? (< nil 5)) true "less nil vs value is unknown")
	(assert (nil? (< 5 nil)) true "less value vs nil is unknown")
	(assert (< 1.5 2.5) true "less float vs float")
	(assert (< 2.5 1.5) false "less float vs float reverse")
	(assert (< "apple" "banana") true "less string vs string")
	(assert (< "banana" "apple") false "less string vs string reverse")
	(assert (< false true) true "less bool false < true")
	(assert (< "1" 2) true "less string vs int coerced")

	/* strlike */
	(assert (strlike "a" "a") true "strlike simple")
	(assert (strlike "a" "_") true "strlike single")
	(assert (strlike "a" "_5") false "strlike overlap")
	(assert (strlike "asdf" "asdf") true "strlike complex")
	(assert (strlike "asdf" "as%") true "strlike prefix")
	(assert (strlike "asdf" "%df") true "strlike postfix")
	(assert (strlike "asdf" "a%f") true "strlike infix")
	(assert (strlike "acdf" "asdf") false "!strlike complex")
	(assert (strlike "abdf" "as%") false "!strlike prefix")
	(assert (strlike "asdfm" "%df") false "!strlike postfix")
	(assert (strlike "masdf" "a%f") false "!strlike infix")
	(assert (strlike "asd whatever mif" "a%ever%f") true "two infix")

	/* match */
	(assert (match '(1 2 3 5 6) (merge '(a b) rest) (concat "a=" a ", b=" b ", rest=" rest)) "a=1, b=2, rest=(3 5 6)" "match merge")

	/* Lists */
	/* count / nth / append / append_unique */
	(assert (equal? (count '(1 2 3)) 3) true "count on list")
	(assert (equal? (nth '(10 20 30) 1) 20) true "nth returns element")
	(assert (equal? (append '(1 2) 3 4) '(1 2 3 4)) true "append extends list")
	(assert (equal? (append_unique '(1 2 2) 2 3) '(1 2 2 3)) true "append_unique keeps first duplicates only")

	/* cons / car / cdr */
	(assert (equal? (cons 1 '(2 3)) '(1 2 3)) true "cons builds list")
	(assert (equal? (car '(9 8 7)) 9) true "car head")
	(assert (equal? (cdr '(9 8 7)) '(8 7)) true "cdr tail")

	/* zip / merge / merge_unique */
	(assert (equal? (zip (list (list 1 2) (list 3 4))) (list (list 1 3) (list 2 4))) true "zip list of lists")
	(assert (equal? (merge (list (list 1 2) (list 3))) '(1 2 3)) true "merge flattens")
	(assert (equal? (merge_unique (list (list 1 2) (list 2 3))) '(1 2 3)) true "merge_unique removes duplicates")
	(assert (equal? (merge_unique_mut '(1 2) '(2 3)) '(1 2 3)) true "merge_unique_mut multi-arg semantics")
	(assert (equal? (merge_unique_mut '(1 1 2) '(2 3)) '(1 2 3)) true "merge_unique_mut deduplicates first arg too")
	(assert (equal? (merge_unique_mut (list (list 1 2) (list 2 3))) '(1 2 3)) true "merge_unique_mut single-arg list-of-lists semantics")

	/* has? / filter / map / mapIndex / reduce */
	(assert (has? '("a" "b" "c") "b") true "has? finds element")
	(assert (equal? (filter '(1 2 3 4) (lambda (x) (> x 2))) '(3 4)) true "filter keeps >2")
	(assert (equal? (map '(1 2 3) (lambda (x) (+ x x))) '(2 4 6)) true "map doubles")
	(assert (equal? (mapIndex '(10 20) (lambda (i v) (string i))) '("0" "1")) true "mapIndex uses index (stringified)")
	(assert (equal? (reduce '(1 2 3 4) (lambda (acc x) (+ acc x)) 0) 10) true "reduce sums")

	/* list? / contains? */
	(assert (list? '(1 2)) true "list? true on list")
	(assert (list? "x") false "list? false on string")
	(assert (contains? '("a" "b") "b") true "contains? present")
	(assert (contains? '("a" "b") "c") false "contains? absent")

	/* queryplan.scm Neumann rebuild contract */
	(import "queryplan.scm")
	(print "testing queryplan Neumann rebuild contract ...")
	(define tblx "t")
	(define expr_gc (list 'get_column "t" false "id" false))
	(assert (match expr_gc '((symbol get_column) (eval tblx) _ col _) col "no") "id" "match get_column for alias t -> id")
	(define expr_gc_src (source "unit" 1 1 expr_gc))
	(assert (match expr_gc_src '((symbol get_column) (eval tblx) _ col _) col "no") "id" "match get_column with SourceInfo wrapper")
	(define simple_select_ast (list "memcp-tests"
		(list (list "t" "memcp-tests" "t" false nil))
		(list "id" expr_gc)
		true nil nil nil nil nil))
	(define simple_ir (untangle_query_term simple_select_ast nil))
	(assert (equal? (ir_kind simple_ir) 'select) true "untangle_query_term returns select IR")
	(assert (equal? (logical_op (ir_root simple_ir)) 'query-block) true "untangle_query builds a combined query-block root")
	(assert (equal? (source_relation (car (qb_sources (ir_root simple_ir)))) "t") true "query-block keeps base relation logical")
	(assert (equal? (ir_output_fields simple_ir) (list "id" expr_gc)) true "untangle_query separates visible output fields")
	(assert (equal? (ir_hidden_fields simple_ir) '()) true "untangle_query keeps hidden/domain fields separate")
	(assert (equal? (ir_context_get (ir_context_of simple_ir) 'compile-budget-ms nil) 1000) true "untangle_query carries compile budget in context")
	(assert (equal? (join_reorder simple_ir) simple_ir) true "join_reorder is an IR-only phase")
	(assert (equal? (logical_op (build_queryplan_term simple_select_ast)) 'scan) true "build_queryplan lowers simple query-block to physical scan")
	(assert (physical_relational_list_collector?
		(list 'sort 'arbitrarily_renamed_rows (list 'lambda '(a b) true)))
		true "physical plan guard rejects structurally sorted Scheme relations")
	(assert (physical_relational_list_collector?
		(list 'slice 'another_renamed_relation 0 10))
		true "physical plan guard rejects structurally sliced Scheme relations")
	(assert (physical_relational_list_collector?
		(list 'merge (list 'map 'renamed_input (list 'lambda '(row) 'row))))
		true "physical plan guard rejects structurally flattened mapped relations")
	(assert (equal? (split_and_terms (combine_where_terms
		(list
			(list (quote and) "a" "b")
			"c"
			(list (quote and) "d" (list (quote and) "e" "f")))
		true)) '("a" "b" "c" "d" "e" "f")) true "combine_where_terms flattens nested AND terms")
	(define test_exists_union_outer_source
		(list "outer_asset" "memcp-tests" "trace_asset" false nil))
	(define test_exists_union_probe
		(list (quote get_column) "outer_asset" false "id" false))
	(define test_exists_union_branch (lambda (alias relation)
		(begin
			(define src (list alias "memcp-tests" relation false nil))
			(make_query_block "memcp-tests" (list src)
				(list "id" (list (quote get_column) alias false "asset_id" false))
				true '() nil '() nil nil '() '() '()))))
	(define test_exists_union_probe_plan (lower_exists_union_probe_expr
		(list test_exists_union_outer_source)
		"outer_asset"
		(list
			(test_exists_union_branch "ref_a" "trace_ref_a")
			(test_exists_union_branch "ref_b" "trace_ref_b")
			(test_exists_union_branch "ref_c" "trace_ref_c"))
		test_exists_union_probe
		'()))
	(assert (equal? (car test_exists_union_probe_plan) 'or) true
		"EXISTS UNION selective lowering emits one n-ary OR root")
	(assert (equal? (count test_exists_union_probe_plan) 4) true
		"EXISTS UNION selective lowering does not build a binary OR chain")
	(assert (reduce (cdr test_exists_union_probe_plan) (lambda (valid branch)
		(and valid (equal? (car branch) 'scan_exists))) true) true
		"EXISTS UNION selective lowering emits one bounded probe per branch")
	(define graph_col (lambda (alias column)
		(list (quote get_column) alias false column false)))
	(define graph_sources (list
		(list "a" "memcp-tests" "graph_a" false nil)
		(list "b" "memcp-tests" "graph_b" false
			(list (quote equal??) (graph_col "a" "id") (graph_col "b" "a_id")))
		(list "c" "memcp-tests" "graph_c" false
			(list (quote equal??) (graph_col "b" "id") (graph_col "c" "b_id")))))
	(define graph_where (list (quote and)
		(list (quote >) (graph_col "a" "score") 10)
		(list (quote equal??)
			(list (quote +) (graph_col "a" "id") (graph_col "b" "id"))
			(graph_col "c" "all_id"))))
	(define graph_block (make_query_block "memcp-tests" graph_sources '() graph_where
		nil nil nil nil nil '() '() '()))
	(define graph_view (extract_join_hypergraph graph_block))
	(assert (equal? (count (qassoc_get graph_view 'nodes '())) 3) true "join hypergraph contains every logical source")
	(assert (equal? (count (qassoc_get graph_view 'locals '())) 1) true "join hypergraph separates local predicates")
	(assert (equal? (count (qassoc_get graph_view 'edges '())) 2) true "join hypergraph extracts binary ON edges")
	(assert (equal? (count (qassoc_get graph_view 'hyperedges '())) 1) true "join hypergraph preserves predicates spanning three aliases")
	(assert (equal? (count (qassoc_get graph_view 'residuals '())) 0) true "join hypergraph has no residual for referenced predicates")
	(assert (equal? (count (filter (qassoc_get graph_view 'edges '()) (lambda (entry)
		(equal? (qassoc_get entry 'origin nil) 'inner-on)))) 2) true "join hypergraph preserves INNER ON provenance")
	(assert (equal? (qb_where graph_block) graph_where) true "join hypergraph extraction leaves WHERE in its query-block")
	(assert (equal? (map (qb_sources graph_block) source_join_expr)
		(map graph_sources source_join_expr)) true "join hypergraph extraction leaves ON predicates on their sources")
	(define outer_graph_sources (list
		(list "a" "memcp-tests" "graph_a" false nil)
		(list "c" "memcp-tests" "graph_c" false nil)
		(list "b" "memcp-tests" "graph_b" true
			(list (quote equal??) (graph_col "a" "id") (graph_col "b" "a_id")))))
	(define outer_graph (extract_join_hypergraph (make_query_block
		"memcp-tests" outer_graph_sources '() true nil nil nil nil nil '() '() '())))
	(define outer_barrier (car (qassoc_get outer_graph 'barriers '())))
	(assert (equal? (qassoc_get outer_barrier 'owner nil) "b") true "join hypergraph identifies the nullable outer-join source")
	(assert (equal? (qassoc_get outer_barrier 'preserved '()) '("a" "c")) true "outer-join barrier preserves the complete left input")
	(assert (equal? (qassoc_get outer_barrier 'references '()) '("a" "b")) true "outer-join barrier records aliases referenced by ON")
	(assert (equal? (qassoc_get (car (qassoc_get outer_graph 'edges '())) 'origin nil) 'outer-on) true
		"join hypergraph preserves OUTER ON provenance")
	(assert (empty_list? (join_order_local_predicates_for_alias
		(list (list (list "b") 0.1 'outer-on "b" (source_join_expr (nth outer_graph_sources 2))))
		"b")) true "reorder never converts an OUTER ON predicate into leaf ownership")
	(define normalized_graph_block (join_optimizer_normalize_inner_joins graph_block))
	(assert (equal? (count (split_and_terms (qb_where normalized_graph_block))) 4) true "inner ON predicates join the logical query-block predicate cloud")
	(assert (equal? (map (qb_sources normalized_graph_block) source_join_expr) '(nil nil nil)) true "normalized inner sources do not retain physical predicate ownership")
	(define normalized_outer_graph (join_optimizer_normalize_inner_joins (make_query_block
		"memcp-tests" outer_graph_sources '() true nil nil nil nil nil '() '() '())))
	(assert (equal? (source_join_expr (nth (qb_sources normalized_outer_graph) 2))
		(source_join_expr (nth outer_graph_sources 2))) true "outer ON predicate remains attached to its semantic barrier")
	(define graph_plan (join_optimizer_plan_segment '() graph_sources graph_sources "a" graph_view))
	(assert (equal? (qassoc_get graph_plan 'strategy nil) 'dphyp) true "small join graphs use DPHyp")
	(assert (equal? (count (qassoc_get graph_plan 'order '())) 3) true "DPHyp covers every logical join source")
	(assert (equal? (car (qassoc_get graph_plan 'tree '())) 'join-node) true "DPHyp records a bushy logical join tree")
	(assert (equal? (cadr (qassoc_get graph_plan 'tree '())) 'inner) true "optimizer trees distinguish inner joins semantically")
	(define test_join_tree_predicates (lambda (tree)
		(match tree
			((symbol join-leaf) _alias) '()
			((symbol join-leaf) _alias predicates) predicates
			((symbol join-node) _kind left right predicates) (merge (list
				(test_join_tree_predicates left)
				(test_join_tree_predicates right)
				predicates))
			_ '())))
	(define graph_plan_predicate_origins
		(map (test_join_tree_predicates (qassoc_get graph_plan 'tree '())) join_order_pred_origin))
	(define test_join_tree_predicates_placed? (lambda (tree)
		(match tree
			((symbol join-leaf) _alias) true
			((symbol join-leaf) alias predicates)
			(reduce predicates (lambda (valid predicate)
				(and valid (equal? (join_order_pred_aliases predicate) (list alias)))) true)
			((symbol join-node) kind left right predicates)
			(begin
				(define left_aliases (join_optimizer_tree_aliases left))
				(define right_aliases (join_optimizer_tree_aliases right))
				(define combined (merge_unique (list left_aliases right_aliases)))
				(and
					(reduce predicates (lambda (valid predicate)
						(and valid (or
							(join_order_predicate_crosses_in? predicate
								left_aliases right_aliases combined)
							(join_order_predicate_owned_by_barrier? predicate kind right)))) true)
					(and (test_join_tree_predicates_placed? left)
						(test_join_tree_predicates_placed? right))))
			_ false)))
	(assert (equal? (count (test_join_tree_predicates (qassoc_get graph_plan 'tree '()))) 4) true
		"logical join tree owns every reorderable predicate exactly once")
	(assert (test_join_tree_predicates_placed? (qassoc_get graph_plan 'tree '())) true
		"logical join predicates sit on the lowest join node crossing their aliases")
	(assert (equal? (count (filter graph_plan_predicate_origins (lambda (origin) (equal? origin 'inner-on)))) 2) true
		"logical join nodes retain INNER ON provenance from the hypergraph")
	(assert (equal? (count (filter graph_plan_predicate_origins (lambda (origin) (equal? origin 'where)))) 2) true
		"logical join nodes and leaves retain WHERE provenance from the hypergraph")
	(define test_join_tree_leaf_predicates (lambda (tree alias)
		(match tree
			((symbol join-leaf) leaf_alias predicates) (if (equal? leaf_alias alias) predicates '())
			((symbol join-node) _kind left right _predicates) (merge (list
				(test_join_tree_leaf_predicates left alias)
				(test_join_tree_leaf_predicates right alias)))
			_ '())))
	(define graph_a_leaf_predicates (test_join_tree_leaf_predicates
		(qassoc_get graph_plan 'tree '()) "a"))
	(assert (equal? (count graph_a_leaf_predicates) 1) true
		"reorder binds a costed single-alias predicate to its logical join leaf")
	(assert (equal? (join_order_pred_expr (car graph_a_leaf_predicates))
		(list (quote >) (graph_col "a" "score") 10)) true
		"logical leaf ownership preserves the complete local predicate expression")
	(define graph_constant_residual (list (quote equal?) 1 1))
	(define graph_residual_block (make_query_block "memcp-tests" graph_sources '()
		(combine_where graph_where graph_constant_residual)
		nil nil nil nil nil '() '() '()))
	(define graph_residual_view (extract_join_hypergraph graph_residual_block))
	(define graph_residual_plan (join_optimizer_plan_segment
		'() graph_sources graph_sources "a" graph_residual_view))
	(assert (equal? (count (qassoc_get graph_residual_view 'residuals '())) 1) true
		"zero-alias conjunct remains a query-block residual")
	(assert (equal? (count (test_join_tree_predicates
		(qassoc_get graph_residual_plan 'tree '()))) 4) true
		"zero-alias residual is not attached to a leaf or join node")
	(assert (contains? (split_and_terms (condition_without_join_tree_predicates
		(qb_where graph_residual_block) (qassoc_get graph_residual_plan 'tree '())))
		graph_constant_residual) true
		"physical residual retains a zero-alias conjunct after consuming tree ownership")
	(define singleton_stage (make_group_stage
		"logical-singleton" (car graph_sources) '() '(1) '() nil '() '() nil nil
		(list
			(list 'purpose 'scalar_single)
			(list 'preserve_empty_domain true)
			(list 'lookup-keys '()))))
	(define singleton_source (list "scalar" "memcp-tests"
		(list 'stage-output "logical-singleton") true true))
	(define singleton_block (make_query_block
		"memcp-tests" (list (car graph_sources) singleton_source) '()
		(list 'equal?? (graph_col "a" "id") (graph_col "scalar" "value"))
		nil nil nil nil nil '() '() '()))
	(define singleton_reordered
		(hybrid_reorder_query_block_using (list singleton_stage) singleton_block))
	(assert (equal? (qassoc_get (qb_facts singleton_reordered) 'join_driver nil) "scalar") true
		"logical reorder drives a guaranteed singleton stage before a larger base source")
	(define dense_join_aliases (map (produceN 14) (lambda (i) (concat "dense_" i))))
	(define dense_join_nodes (mapIndex dense_join_aliases (lambda (i alias) (list alias (+ 100 i)))))
	(define dense_join_predicates (merge (mapIndex dense_join_aliases (lambda (left_index left)
		(map (filter (produceN 14) (lambda (right_index) (> right_index left_index))) (lambda (right_index)
			(list (list left (nth dense_join_aliases right_index)) 0.1)))))))
	(assert (equal? (join_order_choose_strategy 14 false true) 'linearized-dp) true
		"dense medium regular graphs select IKKBZ plus linearized DP in SCM")
	(assert (equal? (join_order_choose_strategy 14 true true) 'goo-dphyp) true
		"dense hypergraphs select GOO with DPHyp subtree optimization in SCM")
	(assert (equal? (join_order_choose_strategy 101 false true) 'goo-linearized-dp) true
		"very large regular graphs select GOO with linearized-DP subtree optimization in SCM")
	(assert (join_order_degree_exceeds_budget? 13 10000) false
		"degree 13 does not prove that the connected-subgraph budget is exceeded")
	(assert (join_order_degree_exceeds_budget? 14 10000) true
		"degree 14 proves that the connected-subgraph budget is exceeded")
	(define small_linearized (join_order_linearized_dp
		(slice dense_join_nodes 0 4)
		(slice dense_join_aliases 0 4)
		(filter dense_join_predicates (lambda (predicate)
			(join_order_set_subset? (join_order_pred_aliases predicate) (slice dense_join_aliases 0 4))))))
	(assert (equal? (join_order_plan_size (car small_linearized)) 4) true
		"SCM IKKBZ and linearized DP construct a complete bushy plan")
	(define small_goo (join_order_goo
		(slice dense_join_nodes 0 4)
		(slice dense_join_aliases 0 4)
		(filter dense_join_predicates (lambda (predicate)
			(join_order_set_subset? (join_order_pred_aliases predicate) (slice dense_join_aliases 0 4))))))
	(assert (equal? (join_order_plan_size small_goo) 4) true
		"SCM GOO greedily constructs a complete bushy plan")
	(define small_hyper_predicates (append
		(filter dense_join_predicates (lambda (predicate)
			(join_order_set_subset? (join_order_pred_aliases predicate) (slice dense_join_aliases 0 4))))
		(list (list (nth dense_join_aliases 0) (nth dense_join_aliases 1) (nth dense_join_aliases 2)) 0.01)))
	(define small_hyper_goo_dp (join_order_goo_dp
		(slice dense_join_nodes 0 4)
		(slice dense_join_aliases 0 4)
		(join_order_prepare_predicates (slice dense_join_aliases 0 4) small_hyper_predicates)
		true))
	(assert (equal? (join_order_plan_size (car small_hyper_goo_dp)) 4) true
		"SCM GOO-DP applies DPHyp to hypergraph subtrees")
	(define graph_reordered_block (hybrid_reorder_query_block graph_block))
	(define graph_reordered_tree (qassoc_get (qb_facts graph_reordered_block) 'join_plan nil))
	(assert (equal?
		(map (qb_sources graph_reordered_block) source_alias)
		(map graph_sources source_alias)) true
		"join_reorder keeps the query-block source catalog in semantic order")
	(assert (equal?
		(map (qb_sources (apply_join_optimizer_plan graph_reordered_block)) source_alias)
		(map graph_sources source_alias)) true
		"physical preparation leaves the semantic source catalog unchanged")
	(assert (equal? (query_block_join_plan graph_reordered_block graph_sources) graph_reordered_tree) true
		"physical lowering reads join order directly from the logical tree")
	(define prepared_source_order (list (nth graph_sources 2) (car graph_sources) (cadr graph_sources)))
	(define prepared_order_block (make_query_block
		"memcp-tests" graph_sources '() graph_where nil nil nil nil nil '() '()
		(list (list 'join_plan (join_optimizer_left_deep_tree prepared_source_order)))))
	(assert (equal? (map (qb_sources prepared_order_block) source_alias) (map graph_sources source_alias)) true
		"physical preparation never reorders the semantic source catalog")
	(assert (equal? (join_optimizer_tree_aliases (query_block_join_plan prepared_order_block graph_sources))
		(map prepared_source_order source_alias)) true
		"physical preparation records specialized traversal only in the join tree")
	(define semantic_outer_tree (join_optimizer_left_deep_tree outer_graph_sources))
	(assert (equal? (cadr semantic_outer_tree) 'left-outer) true "logical join trees preserve left-outer barriers")
	(assert (equal? (join_order_pred_origin (car (nth semantic_outer_tree 4))) 'outer-on) true
		"LEFT OUTER join nodes own their ON predicates")
	(assert (equal? (join_order_pred_owner (car (nth semantic_outer_tree 4))) "b") true
		"LEFT OUTER join predicates retain their barrier owner")
	(define outer_where_expr (list (quote >) (graph_col "b" "score") 0))
	(define outer_where_block (make_query_block "memcp-tests" outer_graph_sources '()
		outer_where_expr nil nil nil nil nil '() '() '()))
	(define outer_where_graph (extract_join_hypergraph outer_where_block))
	(define outer_where_tree (join_order_tree_with_predicates semantic_outer_tree
		(join_optimizer_metadata_costed_predicates outer_graph_sources "a" outer_where_graph
			(source_aliases outer_graph_sources))))
	(define outer_where_node_predicates (nth outer_where_tree 4))
	(assert (empty_list? (test_join_tree_leaf_predicates outer_where_tree "b")) true
		"nullable-side WHERE is not bound to the pre-null-extension scan leaf")
	(assert (equal? (count (filter outer_where_node_predicates (lambda (predicate)
		(equal? (join_order_pred_expr predicate) outer_where_expr)))) 1) true
		"nullable-side WHERE is bound exactly once to its null-extension barrier")
	(assert (test_join_tree_predicates_placed? outer_where_tree) true
		"join predicates use the earliest node with dependencies and null semantics available")
	(define bushy_scan_tree (make_join_optimizer_node 'inner
		(make_join_optimizer_node 'inner (make_join_optimizer_leaf "a") (make_join_optimizer_leaf "b") '())
		(make_join_optimizer_node 'left-outer (make_join_optimizer_leaf "c") (make_join_optimizer_leaf "d") '()) '()))
	(define bushy_scan_sources (list
		(list "a" "memcp-tests" "graph_a" false nil)
		(list "b" "memcp-tests" "graph_b" false nil)
		(list "c" "memcp-tests" "graph_c" false nil)
		(list "d" "memcp-tests" "graph_d" false nil)))
	(define test_physical_scan_signature (lambda (expr)
		(match expr
			((symbol scan) _tx ((symbol table) _schema relation) _filtercols _filterfn _mapcols mapfn _reducefn _init _limit outer)
			(concat relation ":" (string outer) ">" (test_physical_scan_signature mapfn))
			((symbol scan_order) _tx ((symbol table) _schema relation) _filtercols _filterfn _sortcols _sortdirs _brake _offset _limit _mapcols mapfn _reducefn _init outer)
			(concat relation ":" (string outer) ">" (test_physical_scan_signature mapfn))
			(cons head tail) (concat (test_physical_scan_signature head) (test_physical_scan_signature tail))
			_ "")))
	(define bushy_scan_expr (build_join_scan_with_mapper_using_recipe
		"memcp-tests" bushy_scan_sources bushy_scan_tree "a" '() true true '() 0 -1 false nil '() 'pipeline))
	(assert (equal? (test_physical_scan_signature bushy_scan_expr)
		"graph_a:false>graph_b:false>graph_c:false>graph_d:true>") true
		"physical lowering consumes bushy subtree order and its nested LEFT OUTER boundary")
	(define tree_group_stage (make_group_stage
		"tree-group-stage" graph_reordered_block '() '() '() nil '() '() nil nil '()))
	(assert (equal?
		(map (qb_sources (gs_input (apply_join_optimizer_plan_stage tree_group_stage))) source_alias)
		(map graph_sources source_alias)) true
		"physical preparation preserves semantic source catalogs inside group stages")
	(define tree_union (make_union_block
		(quote all) (list graph_reordered_block graph_reordered_block) '() nil nil '()))
	(assert (equal?
		(map (qb_sources (car (union_branches (apply_join_optimizer_plan_node tree_union)))) source_alias)
		(map graph_sources source_alias)) true
		"physical preparation preserves semantic source catalogs inside UNION branches")
	(define malformed_join_plan_block (make_query_block
		"memcp-tests" graph_sources '() graph_where nil nil nil nil nil '() '()
		(list (list 'join_plan (list 'join-leaf "missing")))))
	(assert (try
		(lambda () (begin (apply_join_optimizer_plan malformed_join_plan_block) false))
		(lambda (_e) true)) true
		"physical preparation rejects a join tree that does not cover its source catalog")
	(define probe_outer_column (list (quote get_column) "outer_row" true "id" true))
	(define probe_logical_column (list (quote get_column) "derived_row" false "id" false))
	(define probe_param (symbol "__probe_key_0"))
	(define probe_param_index (scalar_query_probe_param_index
		(list probe_outer_column) (list probe_logical_column) (list probe_param)))
	(assert (equal?
		(rewrite_scalar_query_probe_params probe_param_index probe_outer_column)
		probe_param) true "scalar query probe binds a direct inherited column")
	(assert (equal?
		(rewrite_scalar_query_probe_params probe_param_index (list (quote stage-fixture) (list probe_outer_column)))
		(list (quote stage-fixture) (list probe_param))) true "scalar query probe binds inherited columns throughout stage data")
	(assert (equal?
		(rewrite_scalar_query_probe_params probe_param_index probe_logical_column)
		probe_param) true "scalar query probe binds pre-derived logical aliases")
	(define canonical_group_sum (list (list (quote get_column) "g" false "amount" false) (quote +) 0))
	(define canonical_group_count (list 1 (quote +) 0))
	(define canonical_group_source (list "g" "memcp-tests" "group_values" false nil))
	(define canonical_group_keys (list (list (quote get_column) "g" false "owner_id" false)))
	(define canonical_group_sum_stage (make_group_stage "group-sum" canonical_group_source '()
		canonical_group_keys (list canonical_group_sum) nil '() '() nil nil '()))
	(define canonical_group_count_stage (make_group_stage "group-count" canonical_group_source '()
		canonical_group_keys (list canonical_group_count) nil '() '() nil nil '()))
	(assert (equal?
		(group_stage_cache_relation canonical_group_sum_stage)
		(group_stage_cache_relation canonical_group_count_stage))
		true "group keytable identity excludes aggregate columns")
	(define rebind_child_ag (list 1 (quote +) 0))
	(define rebind_child_stage (make_group_stage
		"rebind-child"
		(list "child" "memcp-tests" "child_source" false nil)
		'() '(1) (list rebind_child_ag) nil '() '() nil nil '()))
	(define rebind_parent_ag (list
		(list (quote scalar_first_probe)
			rebind_child_stage
			(aggregate_col_name rebind_child_ag)
			(list rebind_child_stage))
		(quote +)
		0))
	(define rebind_parent_stage (make_group_stage
		"rebind-parent"
		(list "parent" "memcp-tests" (make_stage_output_relation "rebind-child") false nil)
		'() '(1) (list rebind_parent_ag) nil '() '() nil nil '()))
	(define rebind_fixture (rebind_derived_stages
		"lookup"
		(list rebind_child_stage rebind_parent_stage)))
	(define rebound_parent_stage (stage_by_id
		(nth rebind_fixture 0)
		(stage_merge_lookup (nth rebind_fixture 2) "rebind-parent" nil)))
	(define rebound_parent_ref (rebind_derived_stage_expr rebind_fixture
		(list (quote get_column)
			(exists_stage_alias "rebind-parent")
			false
			(aggregate_col_name rebind_parent_ag)
			false)))
	(assert (not (equal?
		(aggregate_col_name rebind_parent_ag)
		(aggregate_col_name (car (gs_aggregates rebound_parent_stage)))))
		true "derived stage rebinding changes dependent aggregate column hashes")
	(assert (equal?
		(nth rebound_parent_ref 3)
		(aggregate_col_name (car (gs_aggregates rebound_parent_stage))))
		true "derived stage rebinding updates aggregate column handles")
	/* scalar_first_probe_keytable_cost_preferred? safety: an unknown (non-number)
	probe_work_rows must never be treated as "the carrier is cheaper". The shared
	comparison (planner_direct_presence_probe_preferred?) only weighs the two
	costs when both estimates are numbers; without a probe-count estimate, the
	carrier alternative was never actually costed either, so a keytable could be
	built over a source table far larger than any realistic number of direct
	probes would justify. Treating "not known to be direct-preferred" as
	"carrier preferred" -- rather than "unknown, keep the existing safe path" --
	previously caused exactly that regression against a real production table. */
	(define cost_probe_stage (make_group_stage
		"cost-probe-stage"
		(list "cp" "memcp-tests" "cost_probe_source" false nil)
		'() '(1) (list (list 1 (quote +) 0)) nil '() '() nil nil '()))
	(assert (scalar_first_probe_keytable_cost_preferred? cost_probe_stage nil) false
		"keytable cost check refuses the carrier when probe_work_rows is unknown")
	(assert (scalar_first_probe_keytable_cost_preferred? cost_probe_stage false) false
		"keytable cost check refuses the carrier when probe_work_rows is a non-numeric sentinel")
	(assert (scalar_first_probe_keytable_cost_preferred? cost_probe_stage "72") false
		"keytable cost check refuses the carrier when probe_work_rows is a non-numeric string")
	/* planner_recset_carrier_cost / scalar_first_probe_recset_cost_preferred?:
	same "unknown stays unknown" discipline as the keytable check above, plus
	the actual three-way comparison. A RecSet's build cost scales with
	probe_rows (recset_project_join visits the driving side once, however
	large it is); a keytable's dominant cost is its per-driving-row read
	(row_ns, also scaled by probe_rows). For a large probe_rows count RecSet's
	one-pass build must come out cheaper than both alternatives; for a tiny
	probe_rows count (the same "selective driving query" shape
	traced-union-scalar-probes.yaml guards against for the keytable) none of
	the carriers should look worth building over a handful of direct probes. */
	(assert (scalar_first_probe_recset_cost_preferred? cost_probe_stage nil) false
		"recset cost check refuses the carrier when probe_work_rows is unknown")
	(assert (scalar_first_probe_recset_cost_preferred? cost_probe_stage false) false
		"recset cost check refuses the carrier when probe_work_rows is a non-numeric sentinel")
	(assert (planner_cost_better?
		(planner_recset_carrier_cost 20 200000)
		(planner_direct_presence_probe_cost 200000)) true
		"recset carrier beats direct probing at large driving-row counts")
	(assert (planner_cost_better?
		(planner_recset_carrier_cost 20 200000)
		(planner_presence_carrier_cost 20 200000)) true
		"recset carrier beats the keytable carrier at large driving-row counts (no per-row read term)")
	(assert (planner_cost_better?
		(planner_direct_presence_probe_cost 1)
		(planner_recset_carrier_cost 20 1)) true
		"direct probing beats building any carrier for a single accepted row")
	/* scalar_first_probe_keytable_key_index: the stage's key domain may mix a
	true per-outer-row key with session-constant reads (e.g. a permission check
	correlated to the current user). Only the non-session key identifies which
	outer value drives the lookup; with more than one non-session key, or none,
	there is no single row-identity to build a carrier around. */
	(define session_key_stage (make_group_stage
		"session-key-stage"
		(list "sk" "memcp-tests" "session_key_source" false nil)
		(list (list (quote session) "probe_user"))
		(list (list (quote get_column) "sk" false "id" false) (list (quote session) "probe_user"))
		(list (list 1 (quote +) 0)) nil '() '() nil nil '()))
	(define session_key_src (list "sk" "memcp-tests" "session_key_source" false nil))
	(define session_key_keys (list (list (quote get_column) "sk" false "id" false) (list (quote session) "probe_user")))
	(assert (nil? (scalar_first_probe_keytable_key_index session_key_stage session_key_src session_key_keys)) true
		"keytable key index declines when the source's uniqueness cannot be verified (unknown table)")
	(define multi_row_key_stage (make_group_stage
		"multi-row-key-stage"
		(list "mk" "memcp-tests" "multi_row_key_source" false nil)
		'()
		(list (list (quote get_column) "mk" false "a" false) (list (quote get_column) "mk" false "b" false))
		(list (list 1 (quote +) 0)) nil '() '() nil nil '()))
	(assert (nil? (scalar_first_probe_keytable_key_index multi_row_key_stage
		(list "mk" "memcp-tests" "multi_row_key_source" false nil)
		(list (list (quote get_column) "mk" false "a" false) (list (quote get_column) "mk" false "b" false)))) true
		"keytable key index declines when more than one non-session key remains")
	/* planner_row_count_after_selectivity: an unlimited dataset reduce (a bare
	COUNT(*) with no LIMIT and no unique point lookup) still has a knowable
	call count -- its driving source's own row count -- it just isn't bounded
	by a window. Only fall back to the caller-supplied sentinel when the
	source's row count genuinely cannot be determined (not a base table, or
	an unverifiable/fabricated one), matching the same "unknown stays
	unknown" discipline as the keytable cost check above. */
	(define row_count_unknown_src (list "rc" "memcp-tests" "row_count_unknown_source" false nil))
	(assert (equal? (planner_row_count_after_selectivity
		row_count_unknown_src (list row_count_unknown_src) "rc" true nil) nil) true
		"row count after selectivity falls back to the caller's sentinel for an unverifiable source")
	(assert (equal? (planner_row_count_after_selectivity
		row_count_unknown_src (list row_count_unknown_src) "rc" true 1) 1) true
		"row count after selectivity preserves whatever fallback the caller passed in")
	/* query_invariant_presence_stage?/query_invariant_probe_entries_for_stages:
	a presence probe whose lookup key is a session read (not a reference to
	an outer row's column) is invariant for the whole query execution -- it
	is the same building block lower_group_stage_prepare_using now uses to
	bind such a stage once instead of re-probing it per row. */
	(define invariant_probe_stage (make_group_stage
		"invariant-probe-stage"
		(list "ip" "memcp-tests" "invariant_probe_source" false nil)
		(list (list (quote session) "probe_user"))
		(list (list (quote get_column) "ip" false "user" false))
		(list aggregate_count_descriptor)
		nil '() '() nil nil
		(list
			(list (quote purpose) (quote exists))
			(list (quote presence_only) true)
			(list (quote max_needed_per_domain) 1)
			(list (quote physical_max_rows) 1)
			(list (quote on_overflow) (quote ignore))
			(list (quote cardinality_mode) (quote many))
			(list (quote lookup-keys) (list (list (quote session) "probe_user"))))))
	(assert (query_invariant_presence_stage? invariant_probe_stage) true
		"a base-table presence stage whose lookup key is only a session read is query-invariant")
	(define invariant_entries (query_invariant_probe_entries_for_stages (list invariant_probe_stage)))
	(assert (not (empty_list? invariant_entries)) true
		"query_invariant_probe_entries_for_stages finds the eligible stage")
	(assert (not (empty_list? (query_invariant_probe_bindings invariant_entries))) true
		"a non-empty entry list produces a once-bound define lower_group_stage_prepare_using can emit")
	(define correlated_probe_stage (make_group_stage
		"correlated-probe-stage"
		(list "cp2" "memcp-tests" "correlated_probe_source" false nil)
		(list (list (quote get_column) "outer" false "standort" false))
		(list (list (quote get_column) "cp2" false "user" false))
		(list aggregate_count_descriptor)
		nil '() '() nil nil
		(list
			(list (quote purpose) (quote exists))
			(list (quote presence_only) true)
			(list (quote max_needed_per_domain) 1)
			(list (quote physical_max_rows) 1)
			(list (quote on_overflow) (quote ignore))
			(list (quote cardinality_mode) (quote many))
			(list (quote lookup-keys) (list (list (quote get_column) "outer" false "standort" false))))))
	(assert (query_invariant_presence_stage? correlated_probe_stage) false
		"a presence stage whose lookup key references an outer column is not query-invariant -- it must keep using its per-row/keytable probe unchanged")
	(define no_from_select_ast (list "memcp-tests" '() (list "result" 8) true nil nil nil nil nil))
	(assert (equal? (serialize (build_queryplan_term no_from_select_ast))
		"(resultrow '(\"result\" 8))") true "build_queryplan_term lowers no-FROM projection")
	(define inner_derived_ast (list "memcp-tests" '() (list "a" 1) true nil nil nil nil nil))
	(define outer_derived_ast (list "memcp-tests"
		(list (list "d" "memcp-tests" inner_derived_ast false nil))
		(list "*" (list 'get_column nil false "*" false))
		true nil nil nil nil nil))
	(define derived_ir (untangle_query_term outer_derived_ast nil))
	(assert (equal? (qb_sources (ir_root derived_ir)) '()) true "untangle_query flattens zero-source derived table")
	(assert (equal? (ir_output_fields derived_ir) (list "a" 1)) true "untangle_query expands derived-table wildcard projection")
	(assert (equal? (serialize (build_queryplan_term outer_derived_ast))
		"(resultrow '(\"a\" 1))") true "build_queryplan_term lowers flattened zero-source derived table")
	(define scalar_no_from_ast (list "memcp-tests" '()
		(list "x" (list 'inner_select no_from_select_ast)
			"in_ok" (list 'inner_select_in 8 no_from_select_ast)
			"exists_ok" (list 'inner_select_exists no_from_select_ast))
		true nil nil nil nil nil))
	(assert (expr_contains_subquery? (untangle_query_term scalar_no_from_ast nil)) false "untangle_query unnests zero-domain expression subqueries")
	(assert (equal? (serialize (build_queryplan_term scalar_no_from_ast))
		"(resultrow '(\"x\" 8 \"in_ok\" (if (nil? 8) nil (if (nil? 8) nil (equal?? 8 8))) \"exists_ok\" true))") true "build_queryplan_term lowers zero-domain expression subqueries")
	(define nested_catalog_stage (make_group_stage
		"nested-catalog-stage"
		(list "n" "memcp-tests" "nested_source" false nil)
		'() '(1) '() nil '() '() nil nil '()))
	(define nested_catalog_input (make_query_block
		"memcp-tests"
		(list
			(list "o" "memcp-tests" "outer_source" false nil)
			(list "nested" "memcp-tests" (make_stage_output_relation "nested-catalog-stage") false true))
		'() true '() nil '() nil nil '() '() '()))
	(define outer_catalog_stage (make_group_stage
		"outer-catalog-stage"
		nested_catalog_input
		'() '(1) '() nil '() '() nil nil
		(list (list 'stage_catalog (list nested_catalog_stage)))))
	(assert (list? (lower_group_stage_prepare_using (list outer_catalog_stage) (list outer_catalog_stage) outer_catalog_stage))
		true "group-stage lowering keeps nested stage-output metadata from the full catalog")
	(define catalog_root (make_query_block
		"memcp-tests"
		(list (list "o" "memcp-tests" "outer_source" false nil))
		'() true '() nil '() nil nil '() (list outer_catalog_stage) '()))
	(define prepared_catalog_root (ir_root (prepare_physical_queryplan
		(make_ir (quote select) catalog_root '() nil (quote rows)))))
	(assert (count (qb_stages prepared_catalog_root)) 1
		"physical preparation keeps only direct stages on the query block")
	(assert (count (query_block_stage_catalog prepared_catalog_root)) 2
		"physical preparation expands the nested stage catalog once")
	(assert (group_stage? (stage_by_id (query_block_stage_catalog prepared_catalog_root) "nested-catalog-stage"))
		true "prepared physical catalog exposes nested stage metadata for emission")
	(define prepared_stage_lookup (query_block_stage_lookup prepared_catalog_root))
	(assert (lowering_catalog? prepared_stage_lookup) false
		"physical preparation keeps small stage catalogs on the list path")
	(assert (equal? (gs_id (stage_by_id prepared_stage_lookup "nested-catalog-stage")) "nested-catalog-stage")
		true "physical catalog list path resolves stages by ID")
	(define indexed_catalog_stages (cons outer_catalog_stage
		(cons nested_catalog_stage
			(map (produceN 23) (lambda (i)
				(make_group_stage
					(concat "indexed-catalog-stage-" i)
					(list (concat "s" i) "memcp-tests" (concat "indexed_source_" i) false nil)
					'() '(1) '() nil '() '() nil nil '()))))))
	(define prepared_lowering_catalog (make_lowering_catalog indexed_catalog_stages))
	(assert (lowering_catalog? prepared_lowering_catalog) true
		"physical preparation indexes stage catalogs above the measured crossover")
	(assert (lowering_catalog? (make_lowering_catalog (cdr indexed_catalog_stages))) false
		"physical preparation keeps exactly 24 stages on the list path")
	(assert (equal? (gs_id (stage_for_output_relation prepared_lowering_catalog
		(make_stage_output_relation "nested-catalog-stage"))) "nested-catalog-stage")
		true "indexed physical catalog resolves stage-output relations")
	(define outer_catalog_group_cache_source (list
		"group-cache-source"
		(group_stage_cache_schema outer_catalog_stage)
		(group_stage_cache_relation outer_catalog_stage)
		false
		nil))
	(assert (equal? (gs_id (stage_for_group_cache_source prepared_lowering_catalog outer_catalog_group_cache_source))
		"outer-catalog-stage")
		true "indexed physical catalog resolves group-cache sources")
	(define local_catalog_stage (make_group_stage
		"local-catalog-stage"
		(list "local" "memcp-tests" "local_source" false nil)
		'() '(1) '() nil '() '() nil nil '()))
	(define child_lowering_catalog
		(lowering_catalog_with_local_stages prepared_lowering_catalog (list local_catalog_stage)))
	(assert (lowering_catalog? child_lowering_catalog) true
		"local physical stages create a child lowering catalog")
	(assert (equal? (gs_id (stage_by_id child_lowering_catalog "local-catalog-stage")) "local-catalog-stage") true
		"child lowering catalog resolves local stages")
	(assert (equal? (gs_id (stage_by_id child_lowering_catalog "nested-catalog-stage")) "nested-catalog-stage") true
		"child lowering catalog inherits root stage lookup")
	(assert (equal? (gs_id (stage_for_group_cache_source child_lowering_catalog outer_catalog_group_cache_source))
		"outer-catalog-stage") true
		"child lowering catalog inherits root group-cache lookup")
	(assert (count (lowering_catalog_stages child_lowering_catalog)) (+ (count indexed_catalog_stages) 1)
		"child lowering catalog exposes local and inherited stages without changing the root")
	(define extended_list_catalog
		(lowering_catalog_with_local_stages prepared_stage_lookup (list local_catalog_stage)))
	(assert (lowering_catalog? extended_list_catalog) false
		"local physical stages keep a small lowering catalog on the list path")
	(assert (equal? (gs_id (stage_by_id extended_list_catalog "local-catalog-stage")) "local-catalog-stage") true
		"extended list catalog resolves its local stage")
	(assert (equal?
		(stage_dependency_graph indexed_catalog_stages)
		(stage_dependency_graph prepared_lowering_catalog))
		true "indexed physical catalog preserves the dependency graph")
	(assert (count (qassoc_get (gs_facts (car (qb_stages prepared_catalog_root))) (quote stage_catalog) '())) 2
		"small physical stages retain the cache-friendly catalog list")
	(define indexed_catalog_stage (group_stage_with_lowering_catalog outer_catalog_stage prepared_lowering_catalog))
	(assert (empty_list? (qassoc_get (gs_facts indexed_catalog_stage) (quote stage_catalog) '())) true
		"indexed physical stages do not copy the complete prepared catalog into their facts")
	(assert (lowering_catalog? (qassoc_get (gs_facts indexed_catalog_stage) (quote lowering_catalog) nil)) true
		"indexed physical stages retain only the lowering catalog handle")
	(define btw_outer_sources (list (list "o" "memcp-tests" "outer_t" false nil)))
	(define btw_inner_sources (list (list "i" "memcp-tests" "inner_t" false nil)))
	(define btw_outer_id (list 'get_column "o" false "id" false))
	(define btw_outer_limit (list 'get_column "o" false "limit_value" false))
	(define btw_inner_user_id (list 'get_column "i" false "user_id" false))
	(define btw_inner_score (list 'get_column "i" false "score" false))
	(define btw_simple_inner (make_query_block "memcp-tests"
		btw_inner_sources
		(list "score" btw_inner_score)
		(list 'and (list 'equal?? btw_inner_user_id btw_outer_id) (list '> btw_inner_score 0))
		'() nil '() nil nil '() '() '()))
	(assert (equal? (btw2025_query_block_accessing_aliases btw_simple_inner btw_outer_sources) '("o")) true "BTW2025 accessing records simple correlated outer alias")
	(assert (equal? (btw2025_accessing_after_simple btw_simple_inner btw_outer_sources) '()) true "BTW2025 simple D-join elimination removes pure equality dependency")
	(define btw_general_inner (make_query_block "memcp-tests"
		btw_inner_sources
		(list "score" btw_inner_score)
		(list 'and
			(list 'equal?? btw_inner_user_id btw_outer_id)
			(list '< btw_inner_score btw_outer_limit))
		'() nil '() nil nil '() '() '()))
	(assert (equal? (btw2025_query_block_accessing_aliases btw_general_inner btw_outer_sources) '("o")) true "BTW2025 accessing records general correlated outer alias")
	(assert (equal? (btw2025_accessing_after_simple btw_general_inner btw_outer_sources) '("o")) true "BTW2025 accessing after simple keeps non-equality dependency")
	(define btw_exists_rewrite (make_exists_stage_rewrite btw_simple_inner (list btw_outer_sources btw_simple_inner)))
	(define btw_exists_stage (car (nth btw_exists_rewrite 1)))
	(assert (equal? (qassoc_get (gs_facts btw_exists_stage) 'btw2025_accessing '()) '("o")) true "BTW2025 facts expose original accessing set")
	(assert (equal? (qassoc_get (gs_facts btw_exists_stage) 'btw2025_accessing_after_simple '("missing")) '()) true "BTW2025 facts expose simple-eliminated accessing set")
	(assert (equal? (qassoc_get (gs_facts btw_exists_stage) 'btw2025_simple_d_eliminated false) true) true "BTW2025 facts mark successful simple D elimination")
	(assert (equal? (qassoc_get (gs_facts btw_exists_stage) 'btw2025_domain '()) (list btw_outer_id)) true "BTW2025 facts retain domain projection")
	(assert (equal? (qassoc_get (gs_facts btw_exists_stage) 'btw2025_cclasses '()) (list (list btw_inner_user_id btw_outer_id))) true "BTW2025 facts retain equality classes")
	(define btw_expected_repr (list (list btw_outer_id (car (correlation_inner_keys "i" (list (list btw_inner_user_id btw_outer_id)))))))
	(assert (equal? (qassoc_get (gs_facts btw_exists_stage) 'btw2025_repr '()) btw_expected_repr) true "BTW2025 facts retain outer-to-inner representatives")
	(define btw_info (qassoc_get (gs_facts btw_exists_stage) 'btw2025_info nil))
	(assert (equal? (btw2025_info_accessing btw_info) '("o")) true "BTW2025 info exposes original accessing")
	(assert (equal? (btw2025_info_accessing_after_simple btw_info) '()) true "BTW2025 info exposes accessing after simple")
	(define btw_deferred_ctx (make_uctx nil (list (list 'defer-subquery-rewrites true))))
	(define btw_marker_result (untangle_expr_with_stages (list 'inner_select_exists btw_simple_inner) btw_outer_sources btw_deferred_ctx))
	(assert (dependent_subquery_marker? (nth btw_marker_result 0)) true "BTW2025 deferred expr pass leaves dependent marker")
	(define btw_marker_resolved (btw2025_decorrelate_expr_with_stages (nth btw_marker_result 0) nil))
	(assert (expr_contains_subquery? (nth btw_marker_resolved 0)) false "BTW2025 top-down resolver removes dependent marker")
	(define btw_nested_ctx (make_uctx btw_deferred_ctx (list (list 'btw2025-current-handle "djoin:parent"))))
	(define btw_nested_resolved (btw2025_decorrelate_expr_with_stages (nth btw_marker_result 0) btw_nested_ctx))
	(define btw_nested_stage (car (nth btw_nested_resolved 1)))
	(assert (equal? (qassoc_get (gs_facts btw_nested_stage) 'btw2025_parent nil) "djoin:parent") true "BTW2025 top-down resolver links nested dependent joins to their parent")
	(assert (not (equal? (qassoc_get (gs_facts btw_nested_stage) 'btw2025_handle nil) "djoin:parent")) true "BTW2025 top-down resolver gives every dependent join its own handle")

	/* nil tblvar */
	(define expr_gc_nil (list 'get_column nil false "foo" false))
	(assert (match expr_gc_nil '((symbol get_column) nil _ col _) col "no") "foo" "match get_column with nil tblvar -> foo")

	/* ORDER mapping: o = ((get_column t.col) dir) -> extract col */
	(define order1 (list (list 'get_column "t" false "id" false) true))
	(assert (equal? (match order1 '(((symbol get_column) (eval tblx) _ col _) dir) (list col) '()) '("id")) true "match order key extraction")

	/* aggregate head detection */
	(define expr_agg (list 'aggregate 1 '+ 0))
	(assert (match expr_agg (cons (symbol aggregate) args) args "no") '(1 '+ 0) "match aggregate captures args")

	/* star expansion head (tbl.*) */
	(define expr_star (list 'get_column "t" true "*" false))
	(assert (match expr_star '((symbol get_column) (eval tblx) ignorecase "*" _) "ok" "no") "ok" "match tbl.* with case-insensitive flag")

	/* match/matchConcat patterns (scm/match.go coverage) */
	(print "testing match/matchConcat patterns ...")

	/* matchConcat: prefix + variable */
	(assert (match "hello_world" (concat "hello_" rest) rest "no") "world" "matchConcat prefix+var")
	/* matchConcat: variable + suffix */
	(assert (match "foo_bar" (concat prefix "_bar") prefix "no") "foo" "matchConcat var+suffix")
	/* matchConcat: prefix + middle + suffix */
	(assert (match "pre_mid_suf" (concat "pre_" middle "_suf") middle "no") "mid" "matchConcat 3-part split")
	/* matchConcat: var + delim + var (infix) */
	(assert (equal? (match "key.value" (concat mc_left "." mc_right) (concat mc_left ":" mc_right) "no") "key:value") true "matchConcat infix split")
	/* matchConcat: single var captures all */
	(assert (match "everything" (concat mc_x) mc_x "no") "everything" "matchConcat single var")
	/* matchConcat: prefix mismatch */
	(assert (match "xyz" (concat "abc" mc_rest) mc_rest "no") "no" "matchConcat prefix mismatch")
	/* matchConcat: suffix mismatch */
	(assert (match "xyz" (concat mc_pfx "abc") mc_pfx "no") "no" "matchConcat suffix mismatch")
	/* matchConcat: delimiter not found */
	(assert (match "nocolon" (concat mc_l ":" mc_r) "yes" "no") "no" "matchConcat delim not found")
	/* matchConcat: non-string input */
	(assert (match 42 (concat mc_x) mc_x "no") "no" "matchConcat rejects non-string")

	/* match: type-checking patterns */
	(assert (match "hello" (string? mc_s) mc_s "no") "hello" "match string? captures")
	(assert (match 42 (string? mc_s) mc_s "no") "no" "match string? rejects number")
	(assert (match 3.14 (number? mc_n) mc_n "no") 3.14 "match number? captures")
	(assert (match "abc" (number? mc_n) mc_n "no") "no" "match number? rejects string")
	(assert (match (list 10 20 30) (list? mc_l) (count mc_l) "no") 3 "match list? captures list")
	(assert (match "abc" (list? mc_l) mc_l "no") "no" "match list? rejects string")

	/* match: ignorecase */
	(assert (match "Hello" (ignorecase "hello") "yes" "no") "yes" "match ignorecase match")
	(assert (match "Hello" (ignorecase "world") "yes" "no") "no" "match ignorecase mismatch")

	/* match: regex */
	(assert (match "v=5" (regex "^v=(.*)" _ mc_v) mc_v "no") "5" "match regex single capture")
	(assert (match "abc" (regex "^v=(.*)" _ mc_v) mc_v "no") "no" "match regex no match")
	(assert (equal? (match "key=val" (regex "^(.*)=(.*)$" _ mc_k mc_v) (concat mc_k ":" mc_v) "no") "key:val") true "match regex multi-capture")

	/* match: list destructuring */
	(assert (match (list 10 20 30) (list mc_a mc_b mc_c) (+ mc_a mc_b mc_c) "no") 60 "match list destructure")
	(assert (match (list 10 20) (list mc_a mc_b mc_c) "yes" "no") "no" "match list length mismatch")
	(assert (match "notalist" (list mc_a) "yes" "no") "no" "match list rejects non-list")

	/* match: quote/symbol literal */
	(assert (match 'foo (quote foo) "yes" "no") "yes" "match quote literal match")
	(assert (match 'foo (quote bar) "yes" "no") "no" "match quote literal mismatch")
	(assert (match 'bar (symbol bar) "yes" "no") "yes" "match symbol literal match")
	(assert (match 'bar (symbol baz) "yes" "no") "no" "match symbol literal mismatch")
	(assert (match "str" (quote foo) "yes" "no") "no" "match quote rejects non-symbol")
	(assert (match "str" (symbol foo) "yes" "no") "no" "match symbol rejects non-symbol")

	/* match: cons */
	(assert (match (list 10 20 30) (cons mc_h mc_t) mc_h "no") 10 "match cons head")
	(assert (match (list 10 20 30) (cons mc_h mc_t) (count mc_t) "no") 2 "match cons tail length")
	(assert (match (list) (cons mc_h mc_t) "yes" "no") "no" "match cons empty list")

	/* match: literal value matching */
	(assert (match 42 42 "yes" "no") "yes" "match literal int")
	(assert (match "abc" "abc" "yes" "no") "yes" "match literal string")
	(assert (match nil nil "yes" "no") "yes" "match nil symbol")
	(assert (match true true "yes" "no") "yes" "match true symbol")
	(assert (match false false "yes" "no") "yes" "match false symbol")

	/* Tests for scm package */
	/* Tests for alu.go */

	/* Test for number? */
	(assert (number? 42) true "42 should be a number")
	(assert (number? "42") false "\"42\" should not be a number")
	(assert (number? `symbol) false "'symbol' should not be a number")
	(assert (symbol? (symbol "symbol")) true "constructed symbol should be a symbol")
	(assert (symbol? 0) false "integer should not be a symbol")
	(assert (symbol? false) false "boolean should not be a symbol")

	/* Test for int? (requires int64-producing builtin like size/now) */
	(assert (int? (size "abc")) true "size returns an int")
	(assert (int? 42) false "literal 42 is not an int (parsed as number)")

	/* Test for + */
	(assert (+ 1 2) 3 "1 + 2 should be 3")
	(assert (+ 1 2 3) 6 "1 + 2 + 3 should be 6")
	(assert (+ 1 2 3.5) 6.5 "int + float promotes to float")
	(assert (nil? (+ 1 2 nil)) true "+ with nil returns nil")
	(assert (+ 0 0) 0 "0 + 0 should be 0")

	/* Test for - */
	(assert (- 5 3) 2 "5 - 3 should be 2")
	(assert (- 5 3 1) 1 "5 - 3 - 1 should be 1")
	(assert (equal? (- 5 3.5) 1.5) true "int - float promotes to float")
	(assert (nil? (- 5 nil)) true "- with nil returns nil")

	/* Test for * */
	(assert (* 2 3) 6 "2 * 3 should be 6")
	(assert (* 2 3 4) 24 "2 * 3 * 4 should be 24")
	(assert (equal? (* 2 3.5) 7.0) true "int * float promotes to float")
	/* regression: first arg float must not be truncated to int */
	(assert (equal? (* 2.5 2) 5.0) true "float * int (float first) -> 5.0")
	(assert (equal? (* 2.5 2 2) 10.0) true "float-first multiply across ints -> 10.0")
	/* additional: begin with integers, only integers -> stays int */
	(assert (int? (* 2 3 4 5)) true "int*int*int*int stays int type")
	(assert (* 1 2 3 4 5) 120 "many ints multiply correctly")
	/* additional: mixed starting with integers -> promote when float appears */
	(assert (equal? (* 2 3 0.5) 3.0) true "int*int*float -> 3.0 (float)")
	(assert (equal? (* 2 0.5 3) 3.0) true "int*float*int -> 3.0 (float)")
	(assert (equal? (* 2 3 4.0) 24.0) true "int*int*float -> 24.0 (float)")
	/* nil propagation */
	(assert (nil? (* 2 nil)) true "* with nil at end returns nil")
	(assert (nil? (* nil 2 3)) true "* with nil at beginning returns nil")
	(assert (nil? (* 2 3 nil)) true "* with nil at end (3-args) returns nil")
	(assert (nil? (* 2 3.5 nil)) true "* with int->float change then nil returns nil")

	/* Test for / */
	(assert (/ 6 2) 3 "6 / 2 should be 3")
	(assert (/ 12 2 2) 3 "12 / 2 / 2 should be 3")
	(assert (equal? (/ 7 2) 3.5) true "int / int yields float when needed")
	(assert (nil? (/ 5 nil)) true "/ with nil returns nil")

	/* Test for < */
	(assert (< 1 2) true "1 < 2 should be true")
	(assert (< 2 1) false "2 < 1 should be false")

	/* Test for <= */
	(assert (<= 1 2) true "1 <= 2 should be true")
	(assert (<= 2 2) true "2 <= 2 should be true")
	(assert (<= 3 2) false "3 <= 2 should be false")

	/* Test for > */
	(assert (> 2 1) true "2 > 1 should be true")
	(assert (> 1 2) false "1 > 2 should be false")

	/* Test for >= */
	(assert (>= 2 1) true "2 >= 1 should be true")
	(assert (>= 2 2) true "2 >= 2 should be true")
	(assert (>= 1 2) false "1 >= 2 should be false")

	/* Test for equal? */
	(assert (equal? 2 2) true "2 equal? 2 should be true")
	(assert (equal? 2 3) false "2 equal? 3 should be false")

	/* Test for equal?? */
	(assert (equal?? 42 42) true "42 equal?? 42 should be true")
	(assert (equal?? 42 43) false "42 equal?? 43 should be false")
	(assert (equal?? "hello" "HELLO") true "\"hello\" equal?? \"HELLO\" should be true")
	(assert (equal?? "hello" "world") false "\"hello\" equal?? \"world\" should be false")
	(assert (equal?? true true) true "true equal?? true should be true")
	(assert (equal?? true false) false "true equal?? false should be false")

	/* Test for ! */
	(assert (! true) false "not true should be false")
	(assert (! false) true "not false should be true")

	/* Test for not */
	(assert (not true) false "not true should be false")
	(assert (not false) true "not false should be true")

	/* Truthiness of quoted symbols in special forms (no type-enforced not) */
	(assert (if false 1 2) 2 "false treated as falsy in if")
	(assert (if nil 1 2) 2 "'nil treated as falsy in if")

	/* Test for nil? */
	(assert (nil? nil) true "nil? of nil should be true")
	(assert (nil? 0) false "nil? of 0 should be false")

	/* Test for min */
	(assert (equal? (min 1 2 3) 1) true "min of 1, 2, 3 should be 1")
	(assert (equal? (min 5 3 1) 1) true "min of 5, 3, 1 should be 1")

	/* Test for max */
	(assert (equal? (max 1 2 3) 3) true "max of 1, 2, 3 should be 3")
	(assert (equal? (max 5 3 1) 5) true "max of 5, 3, 1 should be 5")

	/* Test for floor */
	(assert (equal? (floor 3.7) 3) true "floor of 3.7 should be 3")
	(assert (equal? (floor 3.2) 3) true "floor of 3.2 should be 3")

	/* Test for ceil */
	(assert (equal? (ceil 3.7) 4) true "ceil of 3.7 should be 4")
	(assert (equal? (ceil 3.2) 4) true "ceil of 3.2 should be 4")

	/* Test for round */
	(assert (equal? (round 3.7) 4) true "round of 3.7 should be 4")
	(assert (equal? (round 3.2) 3) true "round of 3.2 should be 3")

	/* Dictionaries / Assoc lists (with FastDict auto-upgrade) */
	(print "testing dictionaries ...")

	/* small assoc basic ops */
	(define d '())
	(set d (set_assoc d "a" 1))
	(set d (set_assoc d "b" 2))
	(assert (has_assoc? d "a") true "assoc has a")
	(assert (has_assoc? d "x") false "assoc no x")
	(assert (equal? (reduce_assoc d (lambda (acc k v) (+ acc v)) 0) 3) true "reduce sum small")
	(define la (list "a" 1 "b" 2))
	(assert (equal? (la "a") 1) true "call assoc as func(list)")
	(assert (equal? (d "b") 2) true "call assoc as func(dict)")

	/* overwrite should not grow list length */
	(set d (set_assoc d "a" 11))
	(assert (equal? (d "a") 11) true "overwrite list assoc value")
	(assert (equal? (count d) 4) true "list length unchanged on overwrite")

	/* merge + map + filter */
	(define d1 (list "x" 10 "y" 20))
	(define d2 (list "y" 5  "z" 7))
	(define dm (merge_assoc d1 d2))
	(assert (equal? (dm "y") 5) true "merge overwrites second wins")
	(define dmap (map_assoc dm (lambda (k v) (+ v 1))))
	(assert (equal? (dmap "z") 8) true "map increments values")
	(define dkeymap (mapkey_assoc dm (lambda (k v) (concat "key:" k))))
	(assert (equal? (dkeymap "key:x") 10) true "mapkey remaps keys")
	(assert (nil? (dkeymap "x")) true "mapkey removes old key")
	(define dkeycollision (mapkey_assoc (list "a" 1 "b" 2) (lambda (k v) "same")))
	(assert (equal? (dkeycollision "same") 2) true "mapkey collision last wins")
	(define df (filter_assoc dmap (lambda (k v) (> v 10))))
	(assert (has_assoc? df "x") true "filter keeps x")
	(assert (has_assoc? df "z") false "filter drops z")
	(assert (equal? (find_assoc df (lambda (k v) (equal? k "x"))) '("x" 11)) true "find_assoc finds slice pair")
	(assert (equal? (find_assoc df (lambda (k v) (equal? k "missing")) '("fallback" 0)) '("fallback" 0)) true "find_assoc default on slice")

	/* big assoc to test auto switch to FastDict */
	(define big (reduce (produceN 2000) (lambda (acc i) (set_assoc acc (concat "k" i) i)) '()))
	(assert (equal? (reduce_assoc big (lambda (acc k v) (+ acc v)) 0) 1999000) true "reduce sum big (0..1999)")

	/* FastDict getter correctness on many keys */
	(assert (has_assoc? big "k0") true "fastdict has k0")
	(assert (has_assoc? big "k1234") true "fastdict has k1234")
	(assert (equal? (big "k1999") 1999) true "fastdict getter last key")
	(assert (equal? (big "k1") 1) true "fastdict getter small key")

	/* Overwrite existing key in FastDict and get updated value */
	(set big (set_assoc big "k100" 555))
	(assert (equal? (big "k100") 555) true "fastdict overwrite value")

	/* extract_assoc produces all keys (sanity: count) */
	(define countkeys (reduce (extract_assoc big (lambda (k v) 1)) (lambda (a b) (+ a b)) 0))
	(assert (equal? countkeys 2000) true "fastdict extract returns all keys (2000)")

	/* map_assoc and filter_assoc over FastDict */
	(define biginc (map_assoc big (lambda (k v) (+ v 1))))
	(assert (equal? (biginc "k0") 1) true "map fastdict increments")
	(define bigkeys (mapkey_assoc big (lambda (k v) (concat "id:" k))))
	(assert (equal? (bigkeys "id:k100") 555) true "mapkey fastdict remaps keys")
	(define bigf (filter_assoc biginc (lambda (k v) (> v 1000))))
	(assert (has_assoc? bigf "k1500") true "filter keeps large values")
	(assert (has_assoc? bigf "k1") false "filter drops small values")
	(assert (equal? (find_assoc bigf (lambda (k v) (equal? k "k1500"))) '("k1500" 1501)) true "find_assoc finds FastDict pair")

	/* set_assoc immutability: original must not be modified */
	(define orig '("a" 1 "b" 2))
	(define modified (set_assoc orig "a" 99))
	(assert (orig "a") 1 "set_assoc immutable: original unchanged on slice path")
	(assert (modified "a") 99 "set_assoc immutable: modified has new value")
	(define orig_fd (reduce (produceN 20) (lambda (acc i) (set_assoc acc (concat "k" i) i)) '()))
	(define modified_fd (set_assoc orig_fd "k5" 999))
	(assert (orig_fd "k5") 5 "set_assoc immutable: original FastDict unchanged")
	(assert (modified_fd "k5") 999 "set_assoc immutable: modified FastDict has new value")

	/* Strings / JSON */
	(print "testing strings ...")
	(assert (equal? (strlen "abc") 3) true "strlen counts bytes")
	(assert (equal? (replace "a-b-c" "-" ":") "a:b:c") true "replace replaces all")
	(assert (equal? (split "a,b,c" ",") '("a" "b" "c")) true "split splits on sep")
	(assert (strlike (htmlentities "<tag>") "&lt;tag&gt;") true "htmlentities encodes angle brackets")
	(assert (equal? (urldecode (urlencode "a b")) "a b") true "url roundtrip")
	(assert (strlike (json_encode_assoc (list "x" 1)) "%\"x\":1%") true "json_encode_assoc contains key and value")

	/* string? / substr / simplify / case conversion */
	(assert (string? "foo") true "string? on string")
	(assert (string? 123) false "string? on number")
	(assert (equal? (substr "hello" 1 3) "ell") true "substr with length")
	(assert (equal? (substr "hello" 1) "ello") true "substr to end")
	(assert (equal? (simplify "3.14") 3.14) true "simplify numeric string")
	(assert (equal? (simplify "abc") "abc") true "simplify keeps non-numeric")
	(assert (equal? (toLower "ÄBCd") "äbcd") true "toLower handles letters")
	(assert (equal? (toUpper "ÄBCd") "ÄBCD") true "toUpper handles letters")

	/* null byte escape in string literals */
	(assert (equal? (strlen "\0") 1) true "\\0 produces single null byte")
	(assert (equal? "\0" (substr "a\0b" 1 1)) true "\\0 survives concat/substr")
	(assert (equal? (replace "x\0y" "\0" ":") "x:y") true "null byte is replaceable")

	/* collate comparator */
	(define less_bin (collate "bin"))
	(assert (less_bin "a" "b") true "bin collation: a<b")
	(assert ((collate "bin" true) "a" "b") false "bin reverse: a<b -> false")
	/* general_ci heuristic places ASCII before non-ASCII class like leading 'aa' */
	(assert ((collate "general_ci") "z" "aa") true "general_ci: ASCII first")

	/* SQL unescape */
	(assert (equal? (bin2hex (sql_unescape "a\\nb")) "610a62") true "sql_unescape newline")
	(assert (equal? (sql_unescape "a\\'b") "a'b") true "sql_unescape quote")
	(assert (equal? (bin2hex (sql_unescape "a\\0b")) "610062") true "sql_unescape NUL byte present")

	/* json_encode vs json_encode_assoc semantics (master-compatible) */
	(assert (equal? (json_encode '(1 2 3)) "[1,2,3]") true "json_encode lists as arrays")
	(assert (strlike (json_encode_assoc (list "x" 1 "y" 2)) "%\"x\":1%") true "json_encode_assoc has x:1")
	(assert (strlike (json_encode_assoc (list "x" 1 "y" 2)) "%\"y\":2%") true "json_encode_assoc has y:2")

	/* symbol encoding must preserve type marker */
	(assert (equal? (json_encode (symbol "alpha")) "{\"symbol\":\"alpha\"}") true "json_encode(symbol) -> {symbol:\"alpha\"}")
	(assert (strlike (json_encode (lambda (a b) (+ a b))) "%\"symbol\":\"lambda\"%") true "json_encode(lambda ...) contains lambda symbol header")
	(assert (strlike (json_encode_assoc (list "s" (symbol "S"))) "%\"s\":{\"symbol\":\"S\"}%") true "json_encode_assoc preserves symbol values")

	/* json_decode builds assoc list with functional access */
	(assert (equal? ((json_decode "{\"a\":2}") "a") 2) true "json_decode object -> assoc callable by key")
	(assert (equal? (nth (json_decode "[1,2,3]") 1) 2) true "json_decode array -> list indexable with nth")

	/* Optimizer safeguards for eval/import and aliasing in begin */
	/* Case: preserve old binding while overloading after an eval barrier */
	(define http_test_begin (begin
		(define http_handler (lambda (req res) 1))
		(define old_handler http_handler)
		(eval '(print "optimizer eval barrier test"))
		(define http_handler (lambda (req res) (+ (old_handler req res) 1)))
		(http_handler 0 0)
	))
	(assert http_test_begin 2 "handler layering with eval barrier")

	/* Case: forbid inlining an alias to a symbol redefined later in the same begin */
	(define alias_cycle_guard (begin
		(define a 10)
		(define old_a a)
		(define a (+ old_a 5))
		(equal? a 15)
	))
	(assert alias_cycle_guard true "no self-referential aliasing inlining")

	/* hex/bin encode-decode */
	(assert (equal? (bin2hex "AB") "4142") true "bin2hex encodes bytes to hex")
	(assert (equal? (hex2bin "414243") "ABC") true "hex2bin decodes hex to bytes")
	(assert (equal? (hex2bin (bin2hex "Hello")) "Hello") true "hex/bin roundtrip")

	/* base64 encode/decode */
	(assert (equal? (base64_encode "foo") "Zm9v") true "base64_encode encodes correctly")
	(assert (equal? (base64_decode "Zm9v") "foo") true "base64_decode decodes correctly")
	(assert (equal? (base64_decode (base64_encode "Hello, world!")) "Hello, world!") true "base64 roundtrip")

	/* randomBytes properties */
	(assert (equal? (strlen (randomBytes 0)) 0) true "randomBytes 0 length")
	(assert (equal? (strlen (randomBytes 16)) 16) true "randomBytes length 16")
	/* two independently generated strings should differ (overwhelmingly likely) */
	(assert (equal? (randomBytes 32) (randomBytes 32)) false "two random strings must be unequal")

	/* Streams */
	(print "testing streams ...")
	(assert (equal? (concat (streamString "abc")) "abc") true "streamString -> concat")
	(assert (equal? (concat (zcat (gzip (streamString "hello")))) "hello") true "gzip+zcat roundtrip")
	(assert (equal? (concat (xzcat (xz (streamString "xyz")))) "xyz") true "xz+xzcat roundtrip")

	/* Eval and Parser (Any-wrapped) semantics */
	(print "testing eval and parser semantics ...")
	(assert (eval '(+ 2 3)) 5 "eval executes quoted code")
	/* eval of computed code (list-built call) */
	(assert (equal? (eval (list + 1 2 3)) 6) true "eval applies computed list")
	/* eval on parsed code */
	(assert (equal? (eval (scheme "(+ 2 5)" "eval1.scm")) 7) true "eval scheme AST")
	/* serialize -> scheme -> eval roundtrip */
	(assert (equal? (eval (scheme (serialize (scheme "(+ 3 4)" "ser.scm")))) 7) true "serialize/scheme roundtrip")
	/* quote returns literal data */
	(assert (equal? (quote a) 'a) true "quote symbol")
	(assert (equal? '(1 2) (list 1 2)) true "literal list equals built list")
	/* if multi-branch and else */
	(assert (equal? (if false 1 true 2 3) 2) true "if selects first true branch")
	(assert (equal? (if false 1 false 2 3) 3) true "if else branch")
	/* and/or short-circuit evaluation with side-effect check via session */
	(define sc (newsession))
	(sc "a" 0)
	(and false (sc "a" 1))
	(assert (equal? (sc "a") 0) true "and short-circuits second arg")
	(or true (sc "a" 1))
	(assert (equal? (sc "a") 0) true "or short-circuits second arg")
	/* coalesce and coalesceNil */
	(assert (equal? (coalesce nil "" 0 '()) '()) true "coalesce takes last even if falsy")
	(assert (equal? (coalesceNil nil "" 0) "") true "coalesceNil returns first non-nil")
	/* outer evaluates expression in outer environment */
	(define ox_eval 10)
	(assert (equal? (begin (define ox_eval 20) (eval (list 'outer 'ox_eval))) 10) true "outer reads outer var")
	/* begin creates new scope; final value is last expression */
	(assert (equal? (begin (define p 1) (define q (+ p 1)) q) 2) true "begin uses new scope and returns last")
	/* !begin executes in parent env */
	(define xb 10)
	(define rb (begin (define xb 1) (!begin (define xb 2)) xb))
	(assert (equal? rb 2) true "!begin does not create new scope; updates same begin env")
	(assert (equal? xb 10) true "outer env unchanged by !begin inside begin")
	/* undefined symbol lookup yields nil */
	(assert (nil? unknown_var_12345) true "reading unknown symbol yields nil")
	/* simple regex parser returns captured value */
	(define p_regex (parser (define x (regex "[A-Za-z]+" true)) x))
	(assert (equal? (p_regex "Hello") "Hello") true "parser returns regex capture")
	/* atom parser with constant generator returns constant */
	(define p_atom (parser (atom "FOO" true) "ok"))
	(assert (equal? (p_atom "FOO") "ok") true "atom parser returns constant generator")

	/* error cases intentionally omitted in unit tests to avoid compile-time constant folding side-effects */

	/* Lambda / apply_assoc */
	(print "testing lambdas and apply_assoc ...")
	(assert (equal? ((lambda (x y) (+ x y)) 2 3) 5) true "lambda call")
	(assert (equal? (apply_assoc (lambda (x y) (+ x y)) (list "x" 2 "y" 3)) 5) true "apply_assoc maps assoc args")

	/* for loop (functional) */
	(print "testing for loop ...")
	/* Count to 10 with single state var */
	(assert (equal? (for '(0) (lambda (x) (< x 10)) (lambda (x) (list (+ x 1)))) '(10)) true "for increments to 10")
	/* Sum 0..9 with (x sum) state */
	(define for_res (for '(0 0) (lambda (x sum) (< x 10)) (lambda (x sum) (list (+ x 1) (+ sum x)))))
	(assert (equal? (nth for_res 0) 10) true "for final x=10")
	(assert (equal? (nth for_res 1) 45) true "for sum 0..9=45")
	/* condition false initially -> returns init unchanged */
	(assert (equal? (for '(5) (lambda (x) (< x 0)) (lambda (x) (list (+ x 1)))) '(5)) true "for returns init when condition false")

	/* Assoc merge with custom merge function */
	(print "testing assoc merge ...")
	(define m1 (list "x" 1))
	(set m1 (set_assoc m1 "x" 2 (lambda (old new) (+ old new))))
	(assert (equal? (m1 "x") 3) true "set_assoc merge function")
	(define m2 (merge_assoc (list "a" 1) (list "a" 5) (lambda (old new) (+ old new))))
	(assert (equal? (m2 "a") 6) true "merge_assoc merge function")

	/* FastDict vs assoc equality */
	(print "testing dict equality ...")
	(define ld '("k0" 0 "k1" 1 "k2" 2 "k3" 3 "k4" 4 "k5" 5))
	(define dd (reduce (produceN 6) (lambda (acc i) (set_assoc acc (concat "k" i) i)) ()))
	(assert (equal? ld dd) true "list vs dict equal content")

	/* FastDict unwrapping in match patterns */
	(print "testing FastDict match unwrapping ...")
	(define fd2 (reduce (produceN 10) (lambda (acc i) (set_assoc acc (concat "k" i) i)) '()))
	/* list? should unwrap FastDict to a flat pair list; size = 20 */
	(assert (match fd2 (list? xs) (reduce xs (lambda (acc _i) (+ acc 1)) 0) "no") 20 "FastDict list? unwraps to 20 elements")
	/* cons over FastDict-as-list should take first element "k0" */
	(assert (match fd2 (cons first rest) first "no") "k0" "FastDict cons head extracts first key")
	/* verify next key via cons on rest -> "k1" */
	(assert (match fd2 (cons _ rest) (match rest (cons k1 _) k1 "no") "no") "k1" "FastDict cons rest begins with k1")

	/* Optimizer semantics (constant folding, shadowing, set behavior) */
	(print "testing optimizer semantics ...")

	/* Constant folding candidates (boolean/arithmetic inside lambdas) */
	(assert ((lambda () (and (and true) true))) true "const and true -> true")
	(assert ((lambda () (and true false))) false "const and false -> false")
	(assert ((lambda () (+ 1 2 3))) 6 "const arith folds to 6")
	(assert ((lambda () (if (and true (equal? 1 1)) 1 2))) true "const condition -> true")

	/* Setting and calling lambdas via set */
	(assert (begin (set fn (lambda (x) (+ x 1))) (fn 4)) 5 "set lambda then call")
	(assert (begin (set add2 (lambda (a b) (+ a b))) (add2 2 5)) 7 "set 2-param lambda then call")

	/* Optimize should fold constants */
	(assert (optimize '(+ 1 2)) 3 "optimize folds +")
	(define optimize_telemetry_test (newsession))
	(assert (optimize '(+ 1 2) (lambda (stats)
		(begin
			(optimize_telemetry_test "calls" (+ (coalesce (optimize_telemetry_test "calls") 0) 1))
			(optimize_telemetry_test "stats" stats)))) 3 "optimize telemetry preserves result")
	(assert (optimize_telemetry_test "calls") 1 "optimize telemetry callback runs exactly once")
	(assert (> ((optimize_telemetry_test "stats") "compile_ns") 0) true "optimize telemetry reports compile time")
	(assert (optimize '('concat "a" 2)) "a2" "optimize folds string concat")
	(assert (optimize '('and true '(equal? 2 2))) true "optimize folds and/equal")
	(assert (optimize '('begin '('define 'x 4) '(+ 'x 1))) 5 "optimize inlines define use-once")
	(assert (optimize '('and '('and '('and '(> 'LINEITEM.L_QUANTITY 10)) true) '('equal? 1 '('outer 1)))) '(> 'LINEITEM.L_QUANTITY 10) "SQL filter optimization")

	/* Flatten nested + and * (associative operators) */
	(assert (optimize '('+ 1 '('+ 2 3))) 6 "optimize flattens nested + constants")
	(assert (optimize '('* 2 '('* 3 4))) 24 "optimize flattens nested * constants")
	(assert (optimize '('+ 1 '('+ 2 '('+ 3 4)))) 10 "optimize flattens deeply nested +")
	(assert (optimize '('* 1 '('* 2 '('* 3 4)))) 24 "optimize flattens deeply nested *")
	/* - and / must NOT be flattened (non-associative) */
	(assert (- 10 (- 3 1)) 8 "subtraction not flattened: 10-(3-1)=8")
	(assert (/ 100 (/ 10 2)) 20 "division not flattened: 100/(10/2)=20")

	/* Foldable constant-folding: list wrapping/unwrapping roundtrip */
	/* Simple list folding: (list 1 2 3) with all-const args folds at compile time */
	(assert ((eval (optimize '('lambda '('x) '(car '(list 1 2 3))))) 0) 1 "folded list car returns first element")
	(assert ((eval (optimize '('lambda '('x) '(nth '(list 10 20 30) 2)))) 0) 30 "folded list nth returns correct element")
	/* Nested list folding: inner lists must survive double-fold roundtrip */
	(assert ((eval (optimize '('lambda '('x) '(car '(car '(list '(list 1 2) '(list 3 4))))))) 0) 1 "nested folded list: car of car")
	(assert ((eval (optimize '('lambda '('x) '(count '(car '(list '(list 1 2) '(list 3 4))))))) 0) 2 "nested folded list: count of inner")
	(assert ((eval (optimize '('lambda '('x) '(count '(list '(list 1 2) '(list 3 4)))))) 0) 2 "nested folded list: count of outer")
	/* json_decode constant folding: parses JSON array to list at compile time */
	(assert ((eval (optimize '('lambda '('x) '(car '(json_decode "[10,20,30]"))))) 0) 10 "json_decode folded: car of parsed array")
	(assert ((eval (optimize '('lambda '('x) '(count '(json_decode "[1,2,3,4]"))))) 0) 4 "json_decode folded: count of parsed array")
	(assert ((eval (optimize '('lambda '('x) '(nth '(json_decode "[5,6,7]") 1)))) 0) 6 "json_decode folded: nth of parsed array")
	/* Nested json_decode result used by another foldable: the double-fold scenario */
	(assert ((eval (optimize '('lambda '('x) '(count '(list "vector" '(json_decode "[1,2,3]")))))) 0) 2 "double-fold: list wrapping json_decode result")
	/* json_decode with nested arrays: recursive wrapping must preserve structure */
	(assert ((eval (optimize '('lambda '('x) '(count '(json_decode "[[1,2],[3,4]]"))))) 0) 2 "json_decode folded: nested arrays outer count")
	(assert ((eval (optimize '('lambda '('x) '(count '(car '(json_decode "[[1,2],[3,4]]")))))) 0) 2 "json_decode folded: nested arrays inner count")
	/* Mixed const and variable args: list with some constants should NOT fold */
	(assert ((eval (optimize '('lambda '('x) '(car '(list 'x 2 3))))) 42) 42 "list with variable arg: no premature fold")
	(assert ((eval (optimize '('lambda '('x) '(count '(list 'x 2 3))))) 42) 3 "list with variable arg: count still works")

	/* Flatten nested !begin blocks */
	(assert (begin (define xb1 1) (!begin (!begin (set xb1 2) (set xb1 (+ xb1 3))) (set xb1 (* xb1 10))) xb1) 50 "nested !begin flattens correctly")

	/* Lambda params overshadow outer variables */
	(define y 10)
	(assert ((lambda (y) (+ y 1)) 5) 6 "param shadows outer value")
	(assert y 10 "outer y unchanged after lambda")

	/* set should affect current scope/param, not outer */
	(define sx 1)
	(assert ((lambda (sx) (begin (set sx 3) sx)) 7) 3 "set updates local param")
	(assert sx 1 "outer sx unchanged after local set")

	/* Shadowing via inner define does not leak */
	(define dz 2)
	(define dz_inner (begin (define dz 9) dz))
	(assert dz_inner 9 "inner define returns its value")
	(assert dz 2 "outer dz unchanged after inner define")

	/* Use-once inlining safety: begin with unused define should not change result */
	(assert (begin (define tmp_unused 42) 7) 7 "unused define eliminated")

	/* !list optimization: (list ...) passed to NoEscape parameter uses stack allocation */
	(assert ((eval (optimize '('lambda '('a 'b) '(count '(list 'a 'b))))) 10 20) 2 "!list count")
	(assert ((eval (optimize '('lambda '('a 'b) '(car '(list 'a 'b))))) 5 6) 5 "!list car")
	(assert ((eval (optimize '('lambda '('a 'b 'c) '(nth '(list 'a 'b 'c) 1)))) 10 20 30) 20 "!list nth")
	(assert ((eval (optimize '('lambda '('a 'b) '(has? '(list 'a 'b) 'a)))) 5 6) true "!list has?")
	(assert ((eval (optimize '('lambda '('a 'b) '(get_assoc '(list "x" 'a "y" 'b) "x")))) 42 99) 42 "!list get_assoc")
	(assert ((eval (optimize '('lambda '('a 'b) '(cdr '(list 'a 'b))))) 5 6) '(6) "!list cdr")
	(assert ((eval (optimize '('lambda '('a 'b) '(reduce '(list 'a 'b) '('lambda '('acc 'x) '(+ 'acc 'x)) 0)))) 10 20) 30 "!list reduce")

	/* _mut optimization: freshly constructed list triggers in-place operations */
	(assert ((eval (optimize '('lambda '('a 'b 'c) '(map '(list 'a 'b 'c) '('lambda '('x) '(+ 'x 1)))))) 10 20 30) '(11 21 31) "_mut map on fresh list")
	(assert ((eval (optimize '('lambda '('a 'b 'c 'd) '(filter '(list 'a 'b 'c 'd) '('lambda '('x) '(> 'x 2)))))) 1 2 3 4) '(3 4) "_mut filter on fresh list")

	/* Declaration-driven optimizer hooks */
	/* and hook: short-circuit on constant-false, remove constant-true */
	(assert (optimize '('and '(equal? 1 1) '(equal? 1 1) '(equal? 1 2))) false "and hook: short-circuit false")
	(assert (optimize '('and '(equal? 1 1) 'x)) 'x "and hook: removes true constant")
	(assert (optimize '('and '(equal? 1 1) '(equal? 1 1))) true "and hook: all true constants fold")
	(assert (serialize (optimize '('and 'x 'y))) "(and x y)" "and hook: non-const args preserved")
	(assert (serialize (optimize '('and 'x '('and 'y '('and 'z 'w))))) "(and x y z w)" "and hook: flattens nested AND terms")

	/* +/* hooks: associative flattening with symbolic args */
	(assert (serialize (optimize '('+ 'a '('+ 'b 'c)))) "(+ a b c)" "+ hook: flattens nested +")
	(assert (serialize (optimize '('* 'a '('* 'b 'c)))) "(* a b c)" "* hook: flattens nested *")
	(assert (serialize (optimize '('+ 'a '('+ 'b '('+ 'c 'd))))) "(+ a b c d)" "+ hook: deeply nested flatten")

	/* _mut hook via FirstParameterMutable: set_assoc on filter result */
	(assert (serialize (optimize '('set_assoc '('filter '('list) '('lambda '('x) true)) "k" "v"))) "(set_assoc_mut (filter '() (lambda (x) true 1)) \"k\" \"v\")" "_mut hook: set_assoc -> set_assoc_mut")
	/* _mut on append with fresh list arg */
	(assert ((eval (optimize '('lambda '('a 'b) '(append '(list 'a) 'b)))) 10 20) '(10 20) "_mut append on fresh list")
	(assert ((eval (optimize '('lambda '() '(count '(list))))) ) 0 "length hook: count folds empty list")
	(assert ((eval (optimize (list 'lambda '()
		(list 'count (list 'append (list 'list) 1 2 3)))))) 3 "length hook: append preserves empty-base exact length")
	(assert ((eval (optimize (list 'lambda (list 'a 'b 'c)
		(list 'count (list 'append (list 'list 'a) 'b 'c))))) 10 20 30) 3 "length hook: count folds exact append length")
	(assert ((eval (optimize (list 'lambda '()
		(list 'count (list 'map (list 'produceN 4) (list 'lambda (list 'i) 'i))))))) 4 "length hook: count folds propagated map length")
	(assert ((eval (optimize (list 'lambda '()
		(list 'count (list 'mapIndex (list 'produceN 5) (list 'lambda (list 'i 'v) 'i))))))) 5 "length hook: count folds mapIndex over produceN")
	(assert ((eval (optimize (list 'lambda '()
		(list 'count (list 'parallel_map (list 'produceN 3) (list 'lambda (list 'v) 'v))))))) 3 "length hook: count folds parallel_map over produceN")
	(assert ((eval (optimize (list 'lambda '('a 'b 'c)
		(list 'count (list 'cdr (list 'list 'a 'b 'c)))))) 10 20 30) 2 "length hook: cdr preserves exact tail length")
	(assert ((eval (optimize (list 'lambda '('a 'b 'c)
		(list 'count (list 'reverse (list 'list 'a 'b 'c)))))) 10 20 30) 3 "length hook: reverse preserves exact length")
	(assert ((eval (optimize (list 'lambda '()
		(list 'count (list 'extract_assoc (list 'list "a" 1 "b" 2) (list 'lambda (list 'k 'v) 'k))))))) 2 "length hook: extract_assoc preserves exact pair count")
	(assert
		((eval (optimize
			(list 'lambda '()
				(list 'count
					(list 'merge
						(list 'map
							(list 'produceN 3)
							(list 'lambda (list 'i) (list 'list 'i (list '+ 'i 10))))))))))
		6
		"length hook: merge folds map callback list width")
	(assert
		((eval (optimize
			(list 'lambda '()
				(list 'count
					(list 'merge
						(list 'extract_assoc
							(list 'list "a" 1 "b" 2)
							(list 'lambda (list 'k 'v) (list 'list 'k 'v)))))))))
		4
		"length hook: merge folds extract_assoc callback list width")
	(assert
		((eval (optimize
			(list 'lambda '()
				(list 'count
					(list 'merge
						(list 'produceN 4
							(list 'lambda (list 'i) (list 'list 'i (list '* 'i 2))))))))))
		8
		"length hook: merge folds produceN callback list width")
	(assert
		((eval (optimize
			(list 'lambda '()
				(list 'count
					(list 'merge
						(list 'list
							(list 'map (list 'produceN 2) (list 'lambda (list 'i) 'i))
							(list 'extract_assoc (list 'list "a" 1 "b" 2) (list 'lambda (list 'k 'v) 'k)))))))))
		4
		"length hook: merge composes exact producer lengths")
	(assert
		((eval (optimize
			(list 'lambda '('a 'b 'c)
				(list 'count
					(list 'zip
						(list 'list
							(list 'map (list 'produceN 3) (list 'lambda (list 'i) 'i))
							(list 'reverse (list 'list 'a 'b 'c))))))))
			10 20 30)
		3
		"length hook: zip preserves exact producer length")
	(assert
		((eval (optimize
			(list 'lambda '()
				(list 'count (list 'parallelN 4 (list 'lambda (list 'i) 'i)))))))
		4
		"length hook: count folds parallelN length")
	/* scan callback ownership: reduce accumulator enables _mut inside reduce body */
	(assert (serialize (optimize '('scan nil '('table "db" "tbl") '("x") '('lambda '('x) true) '("x") '('lambda '('x) 'x) '('lambda '('acc 'row) '(set_assoc 'acc 'row true)) '(list) nil false))) "(scan nil (table \"db\" \"tbl\") (\"x\") (lambda (x) true 1) (\"x\") (lambda (x) (var 0) 1) (lambda (acc row) (set_assoc_mut (var 0) (var 1) true) 2) '() nil false)" "scan hook: reduce acc enables set_assoc_mut")
	(define opt_merge_unique_ser (serialize (optimize
		(list 'lambda
			(list 'a 'b 'c)
			(list 'merge_unique
				(list 'list 'a 'b)
				(list 'list 'b 'c))))))
	(assert (match opt_merge_unique_ser (regex "merge_unique_mut" _) true false) true "_mut merge_unique on fresh first arg")

	/* merge consumers keep the segment catalog and avoid a flattened temporary. */
	(print "testing optimizer merge consumer fusion ...")
	(define opt_reduce_segments (optimize
		(list 'lambda (list 'parts)
			(list 'reduce (list 'merge 'parts)
				(list 'lambda (list 'acc 'item) (list '+ 'acc 'item)) 0))))
	(assert
		(match (serialize opt_reduce_segments) (regex "reduce_segments" _) true false)
		true
		"optimizer: reduce consumes merge segments directly")
	(define reduce_segments_fn (eval opt_reduce_segments))
	(assert (reduce_segments_fn (list (list 1 2) (list) (list 3 4))) 10 "reduce_segments preserves order and empty segments")

	(define opt_reduce_segments_without_neutral (optimize
		(list 'lambda (list 'parts)
			(list 'reduce (list 'merge 'parts)
				(list 'lambda (list 'acc 'item) (list '+ 'acc 'item))))))
	(define reduce_segments_without_neutral_fn (eval opt_reduce_segments_without_neutral))
	(assert (reduce_segments_without_neutral_fn (list (list) (list 1 2) (list 3))) 6 "reduce_segments preserves implicit first accumulator")
	(assert (reduce_segments_without_neutral_fn (list (list) (list))) nil "reduce_segments preserves empty reduction result")
	(define opt_segment_validation_order (optimize
		(list 'lambda (list 'parts)
			(list 'reduce (list 'merge 'parts)
				(list 'lambda (list 'acc 'item) (list 'panic "callback ran")) 0))))
	(define segment_validation_order_fn (eval opt_segment_validation_order))
	(define segment_validation_error (try
		(lambda () (segment_validation_order_fn (list (list 1) 2)))
		(lambda (e) e)))
	(assert (strlike segment_validation_error "%reduce_segments item%") true "reduce_segments validates every segment before invoking reducer")
	(define opt_dynamic_segment_reducer (optimize
		(list 'lambda (list 'parts 'reducer)
			(list 'reduce (list 'merge 'parts) 'reducer 0))))
	(assert
		(match (serialize opt_dynamic_segment_reducer) (regex "reduce_segments" _) true false)
		false
		"optimizer: dynamic reducer keeps merge validation before reducer lookup")
	(define opt_computed_segment_neutral (optimize
		(list 'lambda (list 'parts 'neutral)
			(list 'reduce (list 'merge 'parts)
				(list 'lambda (list 'acc 'item) (list '+ 'acc 'item))
				(list '+ 'neutral 1)))))
	(assert
		(match (serialize opt_computed_segment_neutral) (regex "reduce_segments" _) true false)
		false
		"optimizer: computed neutral keeps merge validation order")

	/* cons consumes mapped tails without materializing and copying an intermediate list. */
	(print "testing optimizer cons/map fusion ...")
	(define opt_cons_map (optimize
		(list 'lambda (list 'items)
			(list 'cons (list 'quote 'node)
				(list 'map 'items
					(list 'lambda (list 'item) (list 'list (list 'quote 'wrapped) 'item)))))))
	(assert
		(match (serialize opt_cons_map) (regex "cons_map" _) true false)
		true
		"optimizer: cons maps directly into its result")
	(define cons_map_fn (eval opt_cons_map))
	(assert
		(cons_map_fn (list 1 2 3))
		(list 'node (list 'wrapped 1) (list 'wrapped 2) (list 'wrapped 3))
		"cons/map fusion preserves values and order")
	(assert
		(cons_map_fn (list))
		(list 'node)
		"cons/map fusion preserves an empty mapped tail")
	(define cons_map_input (list 4 5 6))
	(cons_map_fn cons_map_input)
	(assert
		cons_map_input
		(list 4 5 6)
		"cons/map fusion does not mutate borrowed input")

	/* owned builder reducers append without copying the growing accumulator. */
	(print "testing optimizer owned list builders ...")
	(define opt_owned_append_builder (optimize
		(list 'lambda (list 'items)
			(list 'reduce 'items
				(list 'lambda (list 'acc 'item)
					(list 'merge 'acc (list 'list 'item)))
				(list 'quote (list))))))
	(assert
		(match (serialize opt_owned_append_builder) (regex "append_mut" _) true false)
		true
		"optimizer: singleton merge appends to owned reducer accumulator")
	(assert
		((eval opt_owned_append_builder) (list 1 2 3 4))
		(list 1 2 3 4)
		"owned reducer append preserves item order")

	(define opt_owned_unique_builder (optimize
		(list 'lambda (list 'items)
			(list 'reduce 'items
				(list 'lambda (list 'acc 'item)
					(list 'if
						(list 'contains? 'acc 'item)
						'acc
						(list 'merge 'acc (list 'list 'item))))
				(list 'quote (list))))))
	(assert
		(match (serialize opt_owned_unique_builder) (regex "append_unique_mut" _) true false)
		true
		"optimizer: contains plus owned append becomes one unique append")
	(assert
		((eval opt_owned_unique_builder) (list 1 2 1 3 2 4))
		(list 1 2 3 4)
		"owned unique builder preserves first occurrence order")

	/* match / match_mut correctness */
	(print "testing match/match_mut correctness ...")
	/* match: literal patterns */
	(assert (match 42 42 "yes" nil) "yes" "match literal int hits branch")
	(assert (match 42 0 "no" "yes") "yes" "match literal int falls through to default")
	(assert (match "hi" "hi" true false) true "match literal string hits branch")
	/* match: cons destructuring */
	(assert (match '(1 2 3) (cons h t) h nil) 1 "match cons: head")
	(assert (match '(1 2 3) (cons h t) t nil) '(2 3) "match cons: tail")
	/* match: nil / empty */
	(assert (match '() '() "empty" "other") "empty" "match nil list")
	(assert (match '(1) '() "empty" "other") "other" "match non-nil falls to default")
	/* match: variable binding */
	(assert (match 7 x (* x 2)) 14 "match binds variable")
	/* match: nested */
	(assert (match '(1 2 3)
		(cons a (cons b rest)) (+ a b)
		nil)
		3 "match nested cons adds head elements")

	/* match_mut: same semantics as match */
	(assert (match_mut 42 42 "yes" nil) "yes" "match_mut literal int hits branch")
	(assert (match_mut '(1 2 3) (cons h t) h nil) 1 "match_mut cons: head")
	(assert (match_mut '(1 2 3) (cons h t) t nil) '(2 3) "match_mut cons: tail")
	(assert (match_mut '() '() "empty" "other") "empty" "match_mut nil list")
	(assert (match_mut 7 x (* x 2)) 14 "match_mut binds variable")
	(assert (match_mut '(1 2 3)
		(cons a (cons b rest)) (+ a b)
		nil)
		3 "match_mut nested cons adds head elements")

	/* match/match_mut agree on all cases */
	(define test_match_agree (lambda (val)
		(equal?
			(match val (cons h t) h 42)
			(match_mut val (cons h t) h 42))))
	(assert (test_match_agree '(1 2 3)) true "match and match_mut agree on cons")
	(assert (test_match_agree '()) true "match and match_mut agree on nil")
	(assert (test_match_agree 99) true "match and match_mut agree on non-list")

	/* optimizer: match_mut insertion */
	(print "testing optimizer match_mut insertion ...")
	/* fresh allocation (list with variable args) → must become match_mut */
	(define opt_fresh_ser (serialize (optimize '('lambda '('x) '('match '('list 'x 2 3) '('cons 'a 'rest) 'a 'x)))))
	(assert (match opt_fresh_ser (regex "match_mut" _) true false) true "optimizer: match on (list x ...) becomes match_mut")
	/* lambda parameter (NthLocalVar after replacement) → must stay match */
	(define opt_param_ser (serialize (optimize '('lambda '('x) '('match 'x nil "nil" "other")))))
	(assert (match opt_param_ser (regex "match_mut" _) true false) false "optimizer: match on lambda param stays match (NthLocalVar has no ownership)")
	/* nested: outer on param stays match, inner on fresh becomes match_mut */
	(define opt_nested_ser (serialize (optimize '('lambda '('x) '('match 'x
		'('cons 'h 't) '('match '('list 'h 't) '('cons 'a 'b) 'a nil)
		nil)))))
	(assert (match opt_nested_ser (regex "match_mut" _) true false) true "optimizer: inner match on fresh list inside lambda becomes match_mut")
	/* reduce callback: accumulator is owned → match on acc becomes match_mut */
	(define opt_reduce_ser (serialize (optimize '('lambda '('lst)
		'('reduce 'lst
			'('lambda '('acc 'x) '('match 'acc '('cons 'h 't) '(+ 'h 'x) 'x))
			'(list))))))
	(assert (match opt_reduce_ser (regex "match_mut" _) true false) true "optimizer: match on owned reduce accumulator becomes match_mut")
	/* cdr is a borrowed slice view; mutating its result must not corrupt the source list. */
	(define filter_cdr_without_aliasing (eval (optimize '('lambda '('xs)
		'('list 'xs '('filter '('cdr 'xs) '('lambda '('x) '(> 'x 2))))))))
	(assert
		(filter_cdr_without_aliasing (list 1 2 3 4))
		(list (list 1 2 3 4) (list 3 4))
		"optimizer: filter after cdr preserves the source list")

	/* REGEXP_REPLACE precompilation: constant pattern gets precompiled */
	(assert ((eval (optimize '('lambda '('s) '(regexp_replace 's "[^0-9]" "")))) "abc123def") "123" "regexp_replace precompilation works")
	(assert ((eval (optimize '('lambda '('s) '(regexp_replace 's "^0+" "")))) "000042") "42" "regexp_replace precompiled strips leading zeros")

	/* Numbered parameter semantics (NthLocalVar / NumVars) */
	(print "testing numbered parameters ...")

	/* Correct case: body uses (var i), NumVars covers indices */
	(define lam_ok '('lambda '('a 'b) '('+ '('var 0) '('var 1)) 2))
	(assert ((eval (optimize lam_ok)) 2 3) 5 "numbered params add correctly (NumVars=2)")

	/* Broken case: body references (var 1) but NumVars too small -> must raise error */
	(define lam_bad '('lambda '('a 'b) '('+ '('var 0) '('var 1)) 1))
	(define panicked (newsession))
	(try (lambda () ((eval (optimize lam_bad)) 2 3)) (lambda (e) (panicked "panic" true)))
	(assert (panicked "panic") true "insufficient NumVars must panic (guards optimizer bug)")

	/* Optimizer regression: lambdas with eval must keep symbol-bound params (no NthLocalVar-only params). */
	(define lam_eval_scope (list
		'lambda
		(list 'session)
		(list
			'begin
			(list 'eval (list 'quote (list 'session "probe" 1)))
			(list 'session "probe")
		)
	))
	(assert (list? lam_eval_scope) true "lambda eval scope fixture must be list AST")
	(define eval_scope_state (newsession))
	(try
		(lambda () (eval_scope_state "opt" (optimize lam_eval_scope)))
		(lambda (e) (panicked "eval-opt-panic" true))
	)
	(assert (panicked "eval-opt-panic") nil "optimizer must not panic on lambda containing eval")
	(define lam_eval_scope_opt (eval_scope_state "opt"))
	(if (panicked "eval-opt-panic") true (begin
		(assert (list? lam_eval_scope_opt) true "optimizer output for lambda AST must remain list AST")
		(if (list? lam_eval_scope_opt)
			(assert (count lam_eval_scope_opt) 3 "lambdas with eval must not be auto-numbered (no NumVars append)")
			true
		)

		(define lam_eval_scope_serialized "")
		(try (lambda () (set lam_eval_scope_serialized (serialize lam_eval_scope_opt))) (lambda (e) (panicked "eval-serialize-panic" true)))
		(assert (panicked "eval-serialize-panic") nil "serialize on optimized lambda must not panic")
		(assert (match lam_eval_scope_serialized (regex "\\(var 0\\)" _) true false) false "lambda with eval must keep session symbol, not (var 0)")
	))

	/* cascade override */
	(print "testing more lambda functions ...")
	(define lam_nested1 (lambda (req res) (+ req res)))
	(define lam_nested2 (lambda (req res) (+ 1 (lam_nested1 req res))))
	(define lam_nested3 (lambda (req res) (lam_nested2 req res)))
	(define lam_nested4 lam_nested3)
	(define lam_nested3 (lambda (req res) (+ 3 (lam_nested4 req res))))
	(assert (lam_nested3 4 7) 15 "nested lambda scope calling")

	/* cascade overrides with same variable name -> value must be drawn into inner scope */
	(set lam_handler (newsession))
	(lam_handler "handler" (lambda (req res) (+ req res)))
	(lam_handler "handler" (begin (set old_handler (lam_handler "handler")) (lambda (req res) (+ 1 (old_handler req res)))))
	(assert ((lam_handler "handler") 4 7) 12 "nested lambda scope overriding")
	(lam_handler "handler" (begin (set old_handler (lam_handler "handler")) (set mid_handler (lambda (req res) (+ 1 (old_handler req res)))) mid_handler))
	(assert ((lam_handler "handler") 4 7) 13 "nested lambda scope overriding with inner variables")

	/* Handler chain via global define (dashboard/sql/rdf pattern) */
	(define test_handler (lambda (x) (concat "base:" x)))
	(define test_handler (begin (set old_th test_handler) (lambda (x) (concat "w1:" (old_th x)))))
	(assert (test_handler "a") "w1:base:a" "global handler chain without workaround")
	(define test_handler (begin (set old_th test_handler) (lambda (x) (concat "w2:" (old_th x)))))
	(assert (test_handler "b") "w2:w1:base:b" "double global handler chain")
	/* Handler chain with if statement (exact dashboard pattern) */
	(define test_handler (lambda (x) (concat "base:" x)))
	(define test_handler (begin (set old_th test_handler) (if true nil) (lambda (x) (concat "wrap:" (old_th x)))))
	(assert (test_handler "c") "wrap:base:c" "handler chain with if-statement")
	/* Handler chain with eval in lambda body */
	(define test_handler (lambda (x) (concat "base:" x)))
	(define test_handler (begin (set old_th test_handler) (if true nil) (lambda (x) (begin (if false (eval '(+ 1 2)) nil) (concat "ev:" (old_th x))))))
	(assert (test_handler "d") "ev:base:d" "handler chain with eval in body")

	/* Mixed-type comparison: be forgiving (SQL) — no panic */
	(try (lambda () (< "x" 1)) (lambda (e) (panicked "cmp-panic" true)))
	(assert (panicked "cmp-panic") nil "mixed-type comparison should not panic")

	/* Sync / Context */
	(print "testing sync/context ...")
	/* newsession key listing */
	(define sess (newsession))
	(sess "a" 1)
	(sess "b" 2)
	(define keys (sess))
	(assert (contains? keys "a") true "session lists key a")
	(assert (contains? keys "b") true "session lists key b")
	/* context + sleep */
	(assert (context (lambda () (sleep 0.005))) true "sleep inside context")
	/* context session */
	(assert (context (lambda () (begin (define s (context "session")) (s "k" 7) (equal? (s "k") 7)))) true "context session set/get")
	/* context check inside valid context */
	(assert (context (lambda () (context "check"))) true "context check returns true in valid context")
	/* once */
	(define once_calls (newsession))
	(once_calls "n" 0)
	(define once_fn (once (lambda (x) (begin (once_calls "n" (+ (once_calls "n") 1)) (+ x 1)))))

	(assert (equal? (once_fn 2) 3) true "once first call computes")
	(assert (equal? (once_fn 99) 3) true "once second call returns cached")
	(assert (equal? (once_calls "n") 1) true "once executes only once")
	/* mutex */
	(define mtx (mutex))
	(assert (equal? (mtx (lambda () 42)) 42) true "mutex executes inner function")

	/* Promise */
	(print "testing promise ...")
	(define p1 (newpromise))
	(assert (nil? (p1 "value")) true "unresolved promise value is nil")
	(assert (nil? (p1 "state")) true "unresolved promise state is nil")
	(define p2 (newpromise))
	(p2 "value" 42)
	(assert (equal? (p2 "value") 42) true "resolved promise returns stored value")
	(assert (equal? (p2 "state") true) true "resolved promise state is true")
	(define p3 (newpromise))
	(p3 "value" 1)
	(p3 "value" 2)
	(assert (equal? (p3 "value") 2) true "second resolution overwrites first")
	(define p4 (newpromise))
	(p4 "value" 5)
	(p4 "fail")
	(assert (equal? (p4 "state") false) true "failed promise state is false")
	(assert (nil? (p4 "value")) true "failed promise without payload clears value")
	(define p5 (newpromise))
	(p5 "fail" "boom")
	(assert (equal? (p5 "state") false) true "failed promise with payload keeps failed state")
	(assert (equal? (p5 "value") "boom") true "failed promise stores payload")
	(define p6 (newpromise))
	(context (lambda () (p6 "value" 99)))
	(assert (equal? (p6 "value") 99) true "promise resolves from inside context")
	(define p7 (newpromise))
	(setTimeout (lambda () (p7 "value" "async")) 1)
	(context (lambda () (sleep 0.02)))
	(assert (equal? (p7 "value") "async") true "promise resolves from async callback")

	/* Scheduler */
	(print "testing scheduler ...")
	(define sched (newsession))
	(sched "done" false)
	(setTimeout (lambda () (sched "done" true)) 1)
	(context (lambda () (sleep 0.01)))
	(assert (sched "done") true "setTimeout fires callback")
	/* clearTimeout */
	(sched "done" false)
	(define tid (setTimeout (lambda () (sched "done" true)) 50))
	(clearTimeout tid)
	(context (lambda () (sleep 0.02)))
	(assert (sched "done") false "clearTimeout cancels callback")
	/* setTimeout with negative delay fires immediately */
	(sched "done" false)
	(setTimeout (lambda () (sched "done" true)) -50)
	(context (lambda () (sleep 0.01)))
	(assert (sched "done") true "setTimeout negative delay fires")
	/* clearTimeout on non-existent ID */
	(assert (clearTimeout 999999) false "clearTimeout non-existent ID returns false")

	/* Date */
	(print "testing date helpers ...")
	(assert (number? (now)) true "now returns number")
	(assert (>= (now) 1000000000) true "now >= 1e9 epoch")

	/* Vectors */
	(print "testing vectors ...")
	(assert (equal? (dot '(1 2 3) '(4 5 6)) 32) true "dot product")
	(assert (equal? (dot '(3 4) '(3 4) "COSINE") 1) true "cosine of identical vectors = 1")
	(assert (equal? (round (* 1000 (dot '(3 4) '(3 4) "EUCLIDEAN"))) 5000) true "euclidean length sqrt(sum) *1000")

	/* JIT compilation */
	(settings "LogJIT" false)
	(print "testing JIT compilation ...")

	/* Native pipeline validation */
	(assert ((jit (lambda (a) a)) 5) 5 "jit: identity lambda native")
	(assert ((jit (lambda () 42))) 42 "jit: constant body native")
	(assert ((jit (lambda (a b) b)) 1 7) 7 "jit: return 2nd param native")
	(assert ((jit (lambda (a b c) c)) 1 2 9) 9 "jit: return 3rd param native")
	(assert ((jit (lambda (a b) a)) 3 4) 3 "jit: return 1st of 2 params native")
	(assert ((jit (lambda (a b c) b)) 1 8 3) 8 "jit: return 2nd of 3 params native")
	(define jit_id_desc (jit (lambda (x) x)))
	(jit-warn-if-fallback jit_id_desc "jit: identity lambda")
	(jit-warn-if-fallback (jit (lambda () 42)) "jit: constant lambda")
	(assert ((jit (lambda () (jit-enabled?)))) (jit-enabled?) "jit-enabled? matches build feature inside jit")
	(define jit_add_desc (jit (lambda (a b) (+ a b))))
	(assert (strlike (serialize jit_add_desc) "(lambda %") true "jit descriptor serializes as lambda")
	(assert (equal? (eval (list jit_add_desc 2 5)) 7) true "jit descriptor executable via eval")
	(assert (equal? (apply jit_add_desc '(2 5)) 7) true "jit descriptor executable via apply")
	(assert (jit? (lambda (a b) (+ a b))) false "jit?: plain lambda reports false")
	(define jit_now_desc (jit (lambda () (now))))
	(jit-warn-if-fallback jit_now_desc "jit: now callback")
	(assert (number? (jit_now_desc)) true "jit: Go callback result remains executable")

	/* Proc identity and environment stay available across native compilation. */
	(define jit_dynamic_value 40)
	(define jit_dynamic_reader (jit (eval '('lambda '() 'jit_dynamic_value))))
	(jit-warn-if-fallback jit_dynamic_reader "jit: dynamic environment read")
	(assert (jit_dynamic_reader) 40 "jit: reads an unresolved symbol from its lexical environment")
	(set jit_dynamic_value 41)
	(assert (jit_dynamic_reader) 41 "jit: resolves a changed lexical binding at call time")
	(set jit_dynamic_value nil)
	(assert (nil? (jit_dynamic_reader)) true "jit: dynamic symbol preserves nil")
	(set jit_dynamic_value false)
	(assert (jit_dynamic_reader) false "jit: dynamic symbol preserves bool")
	(set jit_dynamic_value -17)
	(assert (jit_dynamic_reader) -17 "jit: dynamic symbol preserves int")
	(set jit_dynamic_value 3.25)
	(assert (jit_dynamic_reader) 3.25 "jit: dynamic symbol preserves float")
	(set jit_dynamic_value "dynamic string")
	(assert (jit_dynamic_reader) "dynamic string" "jit: dynamic symbol preserves string")
	(set jit_dynamic_value (quote dynamic-symbol-value))
	(assert (jit_dynamic_reader) (quote dynamic-symbol-value) "jit: dynamic symbol preserves symbol")
	(set jit_dynamic_value (list 1 "two" false))
	(assert (jit_dynamic_reader) '(1 "two" false) "jit: dynamic symbol preserves list")
	(set jit_dynamic_value (parse_date "2026-08-15"))
	(assert (jit_dynamic_reader) (parse_date "2026-08-15") "jit: dynamic symbol preserves date")
	(set jit_dynamic_value +)
	(assert (apply (jit_dynamic_reader) '(4 5)) 9 "jit: dynamic symbol preserves native function")
	(set jit_dynamic_value (lambda (value) (+ value 1)))
	(assert (apply (jit_dynamic_reader) '(6)) 7 "jit: dynamic symbol preserves Scheme Proc")
	/* Self-evaluating values retain their complete Scmer representation. */
	(define jit_literal_string (jit (lambda () "literal string")))
	(define jit_literal_symbol (jit (lambda () (quote literal-symbol))))
	(define jit_literal_list (jit (lambda () '(1 "two" false))))
	(define jit_literal_date_value (parse_date "2026-08-15"))
	(define jit_literal_date (jit (eval (list (quote lambda) '() jit_literal_date_value))))
	(define jit_literal_native_fn (jit (eval (list (quote lambda) '() +))))
	(define jit_literal_scheme_fn_value (lambda (value) (+ value 1)))
	(define jit_literal_scheme_fn (jit (eval (list (quote lambda) '() jit_literal_scheme_fn_value))))
	(jit-warn-if-fallback jit_literal_string "jit: literal string return")
	(jit-warn-if-fallback jit_literal_symbol "jit: literal symbol return")
	(jit-warn-if-fallback jit_literal_list "jit: literal list return")
	(jit-warn-if-fallback jit_literal_date "jit: literal date return")
	(jit-warn-if-fallback jit_literal_native_fn "jit: literal native function return")
	(jit-warn-if-fallback jit_literal_scheme_fn "jit: literal Scheme function return")
	(assert (jit_literal_string) "literal string" "jit: returns a string literal")
	(assert (jit_literal_symbol) (quote literal-symbol) "jit: returns a symbol literal")
	(assert (jit_literal_list) '(1 "two" false) "jit: returns a list literal")
	(assert (jit_literal_date) jit_literal_date_value "jit: returns a date literal")
	(assert (apply (jit_literal_native_fn) '(4 5)) 9 "jit: returns a native function literal")
	(assert (apply (jit_literal_scheme_fn) '(6)) 7 "jit: returns an interpreted Proc literal")
	(define jit_scheme_callee (lambda (value) (+ value 1)))
	(define jit_scheme_caller (jit (lambda (value) (jit_scheme_callee value))))
	(jit-warn-if-fallback jit_scheme_caller "jit: call interpreted Scheme Proc")
	(assert (jit_scheme_caller 6) 7 "jit: native Proc calls an interpreted Scheme Proc")
	(set jit_scheme_callee (jit jit_scheme_callee))
	(jit-warn-if-fallback jit_scheme_callee "jit: Scheme callee")
	(assert (jit_scheme_caller 8) 9 "jit: native Proc follows rebinding to a native Scheme Proc")
	(assert (jit_scheme_callee 10) 11 "jit: Scheme invokes a native Proc")
	(define jit_dynamic_many_callee (lambda (a b c d) (list a b c d)))
	(define jit_dynamic_many_caller (jit (eval '('lambda '('a 'b 'c 'd)
		'('jit_dynamic_many_callee 'a 'b 'c 'd)))))
	(define jit_dynamic_many_list_callee jit_dynamic_many_callee)
	(jit-warn-if-fallback jit_dynamic_many_caller "jit: dynamic call with stack arguments")
	(assert (jit_dynamic_many_caller 1 "two" false (list 4 5)) (list 1 "two" false (list 4 5))
		"jit: dynamic call preserves more than two mixed arguments")
	(set jit_dynamic_many_callee +)
	(assert (jit_dynamic_many_caller 1 2 3 4) 10
		"jit: dynamic call follows rebinding to a native function")
	(set jit_dynamic_many_callee (jit jit_dynamic_many_list_callee))
	(assert (jit_dynamic_many_caller 6 7 8 9) '(6 7 8 9)
		"jit: dynamic call follows rebinding to a native Scheme Proc")
	(define jit_nested_maker (jit (lambda (offset)
		(lambda (value) (+ value offset)))))
	(define jit_nested_add7 (jit_nested_maker 7))
	(assert (jit? jit_nested_add7) (jit-enabled?) "jit: captured nested lambda is compiled")
	(assert (jit_nested_add7 5) 12 "jit: captured nested lambda reads its outer value")
	(define jit_scan_filter (jit (lambda (value) (> value 1))))
	(define jit_scan_map (jit (lambda (value) value)))
	(define jit_scan_reduce (jit (lambda (total value) (+ total value))))
	(jit-warn-if-fallback jit_scan_filter "jit: scan filter callback")
	(jit-warn-if-fallback jit_scan_map "jit: scan map callback")
	(jit-warn-if-fallback jit_scan_reduce "jit: scan reduce callback")
	(assert (scan nil (list (list "value" 1) (list "value" 2) (list "value" 3))
		'("value") jit_scan_filter '("value") jit_scan_map jit_scan_reduce 0)
		5 "jit: storage scan executes compiled filter/map/reduce callbacks")
	(define jit_pointer_callee (lambda (value) (list value "rooted")))
	(define jit_pointer_flow (jit (lambda (value) (list (jit_pointer_callee value) (now)))))
	(jit-warn-if-fallback jit_pointer_flow "jit: rooted callback result")
	(assert (nth (nth (jit_pointer_flow 12) 0) 0) 12 "jit: callback pointer result survives a later callback")
	(define jit_stackmap_flow (jit (lambda (value) (list (concat value "-rooted") (now)))))
	(jit-warn-if-fallback jit_stackmap_flow "jit: stack-map pointer result")
	(assert (nth (jit_stackmap_flow "value") 0) "value-rooted"
		"jit: stack-map root survives a later allocating callback")

	/* Borrowed Go-slice headers stay in the native pipeline as ptr/len/cap. */
	(define jit_list_nth (jit (lambda (xs i) (nth xs i))))
	(define jit_list_car (jit (lambda (xs) (car xs))))
	(define jit_list_cdr (jit (lambda (xs) (cdr xs))))
	(define jit_list_cadr (jit (lambda (xs) (cadr xs))))
	(define jit_list_predicate (jit (lambda (xs) (list? xs))))
	(define jit_list_build2 (jit (lambda (a b) (list a b))))
	(define jit_list_build6 (jit (lambda (a b c d e f) (list a b c d e f))))
	(define jit_list_computed (jit (lambda (a b) (list (+ a b) b))))
	(define jit_list_reordered (jit (lambda (a b) (list b a))))
	(define jit_list_subset (jit (lambda (a b) (list b))))
	(define jit_list_stack_nth (jit (lambda (a b) (nth (list a b) 1))))
	(jit-warn-if-fallback jit_list_nth "jit: nth")
	(jit-warn-if-fallback jit_list_car "jit: car")
	(jit-warn-if-fallback jit_list_cdr "jit: cdr")
	(jit-warn-if-fallback jit_list_cadr "jit: cadr")
	(jit-warn-if-fallback jit_list_predicate "jit: list predicate")
	(jit-warn-if-fallback jit_list_build2 "jit: two-element list constructor")
	(jit-warn-if-fallback jit_list_build6 "jit: six-element list constructor")
	(jit-warn-if-fallback jit_list_computed "jit: computed list constructor")
	(jit-warn-if-fallback jit_list_reordered "jit: reordered list constructor")
	(jit-warn-if-fallback jit_list_subset "jit: subset list constructor")
	(jit-warn-if-fallback jit_list_stack_nth "jit: stack-list nth")
	(assert (jit_list_nth '(10 20 30) 1) 20 "jit: nth reads a Go-slice element")
	(assert (jit_list_car '(10 20 30)) 10 "jit: car reads the first Go-slice element")
	(assert (jit_list_cdr '(10 20 30)) '(20 30) "jit: cdr returns a borrowed Go-slice view")
	(assert (jit_list_cdr '()) '() "jit: cdr handles an empty slice")
	(assert (jit_list_cadr '(10 20 30)) 20 "jit: cadr reads the second Go-slice element")
	(assert (jit_list_predicate '(10 20 30)) true "jit: list? accepts a list")
	(assert (jit_list_predicate 10) false "jit: list? rejects a scalar")
	(assert (jit_list_build2 10 "x") '(10 "x") "jit: list constructor preserves mixed values")
	(assert (nth (jit_list_build2 10 "x") 1) "x" "jit: nth reads an element from a freshly allocated list")
	(assert (jit_list_build6 1 2 3 4 5 6) '(1 2 3 4 5 6) "jit: list constructor preserves six values")
	(assert (jit_list_computed 4 7) '(11 7) "jit: list constructor preserves computed values")
	(assert (jit_list_reordered 4 7) '(7 4) "jit: reordered list constructor materializes in requested order")
	(assert (jit_list_subset 4 7) '(7) "jit: subset list constructor does not expose unrelated inputs")
	(assert (jit_list_stack_nth 4 7) 7 "jit: nth reads an element from an optimizer stack list")
	(define jit_list_source_args (list 1 2))
	(define jit_list_allocated (apply jit_list_build2 jit_list_source_args))
	(assert (nth (nth_mut jit_list_allocated 0 99) 0) 99 "jit: a constructed list is mutable by its owner")
	(assert jit_list_source_args '(1 2) "jit: list allocation does not alias apply arguments")
	(assert (try (lambda () (jit_list_car '())) (lambda (e) "caught")) "caught" "jit: car bounds panic is recoverable")
	(assert (try (lambda () (jit_list_nth '(10) 2)) (lambda (e) "caught")) "caught" "jit: nth bounds panic is recoverable")

	/* Basic arithmetic with single parameter */
	(assert ((jit (lambda (x) (+ x 1))) 4) 5 "jit: x + 1")
	(assert ((jit (lambda (x) (- x 3))) 10) 7 "jit: x - 3")
	(assert ((jit (lambda (x) (* x 2))) 5) 10 "jit: x * 2")

	/* Two parameters */
	(assert ((jit (lambda (a b) (+ a b))) 3 4) 7 "jit: a + b")
	(assert ((jit (lambda (a b c) (+ a b c))) (strlen "a") (strlen "bb") (strlen "ccc")) 6 "jit: a + b + c (3 args int path)")
	(assert (int? ((jit (lambda (a b c) (+ a b c))) (strlen "a") (strlen "bb") (strlen "ccc"))) true "jit: 3-arg + stays int")
	(assert ((jit (lambda (a b c d e) (+ a b c d e))) (strlen "a") (strlen "bb") (strlen "ccc") (strlen "dddd") (strlen "eeeee")) 15 "jit: a + b + c + d + e (5 args int path)")
	(assert (int? ((jit (lambda (a b c d e) (+ a b c d e))) (strlen "a") (strlen "bb") (strlen "ccc") (strlen "dddd") (strlen "eeeee"))) true "jit: 5-arg + stays int")
	(assert ((jit (lambda (x) (+ (strlen "ab") x))) (strlen "abc")) 5 "jit: 2 + x int source fast path")
	(assert (int? ((jit (lambda (x) (+ (strlen "ab") x))) (strlen "abc"))) true "jit: 2 + x result stays int")
	(assert ((jit (lambda (a b) (* a b))) 3 4) 12 "jit: a * b")
	(assert ((jit (lambda (a b) (- a b))) 10 3) 7 "jit: a - b")

	/* Nested operations */
	(assert ((jit (lambda (x) (* (+ x 1) 2))) 4) 10 "jit: (x+1)*2")
	(assert ((jit (lambda (x) (+ (* x 2) 1))) 4) 9 "jit: x*2+1")
	(assert ((jit (lambda (a b c) (+ (* a b) c))) 3 4 5) 17 "jit: a*b+c")
	(assert ((jit (lambda (a b c d) (+ (* a b) (* c d)))) 2 3 4 5) 26 "jit: a*b + c*d")
	(assert ((jit (lambda (x) (+ (+ (+ x 1) 2) 3))) 10) 16 "jit: deeply nested +")
	(assert ((jit (lambda (x) (* (* (* x 2) 2) 2))) 3) 24 "jit: deeply nested *")
	(define jit_nested_count_result (jit (lambda (values) (+ (count values) 1))))
	(define jit_nested_equal_result (jit (lambda (left right) (not (equal? left right)))))
	(jit-warn-if-fallback jit_nested_count_result "jit: nested pointer-free integer result")
	(jit-warn-if-fallback jit_nested_equal_result "jit: nested pointer-free boolean result")
	(assert (jit_nested_count_result '(2 4 6)) 4 "jit: nested count feeds arithmetic natively")
	(assert (jit_nested_equal_result "left" "right") true "jit: nested equality feeds boolean logic natively")
	(assert ((jit (lambda (a b) (- (* a a) (* b b)))) 5 3) 16 "jit: a²-b²")
	(assert ((jit (lambda (a b c) (+ a (- b c)))) 10 7 3) 14 "jit: a+(b-c)")

	/* Comparisons */
	(assert ((jit (lambda (x) (< x 10))) 5) true "jit: x < 10 (true)")
	(assert ((jit (lambda (x) (< x 10))) 15) false "jit: x < 10 (false)")
	(assert ((jit (lambda (x) (> x 0))) 5) true "jit: x > 0")
	(assert ((jit (lambda (a b) (equal? a b))) 5 5) true "jit: a == b (true)")
	(assert ((jit (lambda (a b) (equal? a b))) 5 6) false "jit: a == b (false)")

	/* Conditionals */
	(assert ((jit (lambda (x) (if (< x 5) 1 0))) 3) 1 "jit: if x<5 then 1 else 0 (true)")
	(assert ((jit (lambda (x) (if (< x 5) 1 0))) 7) 0 "jit: if x<5 then 1 else 0 (false)")
	(assert ((jit (lambda () (if true 1 2)))) 1 "jit: true?1:2")
	(assert ((jit (lambda () (or false true)))) true "jit: (or false true)")
	(assert ((jit (lambda () (and true true)))) true "jit: (and true true)")
	(assert (nil? ((jit (lambda () (coalesce))))) true "jit: coalesce empty -> nil")
	(assert ((jit (lambda () (coalesce nil false 0 "")))) "" "jit: coalesce all falsy -> last")
	(assert ((jit (lambda () (coalesce nil false 5 9)))) 5 "jit: coalesce first truthy")
	(assert ((jit (lambda (x) (coalesce nil x 7))) 0) 7 "jit: coalesce dynamic falsy -> fallback arg")
	(assert ((jit (lambda (x) (coalesce nil x 7))) 3) 3 "jit: coalesce dynamic truthy -> x")
	(assert (nil? ((jit (lambda () (coalesceNil))))) true "jit: coalesceNil empty -> nil")
	(assert ((jit (lambda () (coalesceNil nil nil "" 0)))) "" "jit: coalesceNil first non-nil (falsy allowed)")
	(assert ((jit (lambda (x) (coalesceNil nil x 7))) nil) 7 "jit: coalesceNil nil -> fallback arg")
	(assert ((jit (lambda (x) (coalesceNil nil x 7))) 0) 0 "jit: coalesceNil non-nil falsy kept")

	/* Constants */
	(assert ((jit (lambda () 42))) 42 "jit: constant return")
	(assert ((jit (lambda (x) 99)) 5) 99 "jit: ignore param, return constant")

	/* Boolean logic */
	(assert ((jit (lambda (x) (and (> x 0) (< x 10)))) 5) true "jit: and (true)")
	(assert ((jit (lambda (x) (and (> x 0) (< x 10)))) 15) false "jit: and (false)")
	(assert ((jit (lambda (x) (or (< x 0) (> x 10)))) 5) false "jit: or (false)")
	(assert ((jit (lambda (x) (or (< x 0) (> x 10)))) 15) true "jit: or (true)")
	(assert ((jit (lambda (x) (not (< x 5)))) 3) false "jit: not (false)")
	(assert ((jit (lambda (x) (not (< x 5)))) 7) true "jit: not (true)")
	(assert ((jit (lambda (x) (! x))) true) false "jit: ! true -> false")
	(assert ((jit (lambda (x) (! x))) false) true "jit: ! false -> true")
	(assert ((jit (lambda (x) (floor x))) 3.9) 3.0 "jit: floor positive")
	(assert ((jit (lambda (x) (floor x))) -3.1) -4.0 "jit: floor negative")
	(assert ((jit (lambda (x) (ceil x))) 3.1) 4.0 "jit: ceil positive")
	(assert ((jit (lambda (x) (ceil x))) -3.9) -3.0 "jit: ceil negative")
	(assert ((jit (lambda (x) (sql_abs x))) -7) 7 "jit: sql_abs int")
	(assert ((jit (lambda (x) (sql_abs x))) -7.5) 7.5 "jit: sql_abs float")
	(assert (nil? ((jit (lambda (x) (sql_abs x))) nil)) true "jit: sql_abs nil")
	(assert ((jit (lambda (x) (sqrt x))) 9) 3.0 "jit: sqrt positive")
	(assert (nil? ((jit (lambda (x) (sqrt x))) -1)) true "jit: sqrt negative -> nil")
	(assert (nil? ((jit (lambda (x) (sqrt x))) nil)) true "jit: sqrt nil")
	(assert ((jit (lambda (x) (list? x))) '(1 2 3)) true "jit: list? true")
	(assert ((jit (lambda (x) (list? x))) 42) false "jit: list? false")

	/* String operations */
	(assert ((jit (lambda (s) (strlen s))) "hello") 5 "jit: strlen")
	(assert ((jit (lambda (s) (strlen s))) "") 0 "jit: strlen empty")
	(assert ((jit (lambda (s) (strlen s))) "äöü") 6 "jit: strlen utf8 bytes")
	(assert ((jit (lambda (a) (+ (strlen a) 1))) "hello") 6 "jit: + (strlen a) 1")
	(assert ((jit (lambda (s) (string? s))) "hello") true "jit: string? true")
	(assert ((jit (lambda (s) (string? s))) 123) false "jit: string? false")
	/* (assert ((jit (lambda (a b) (concat a b))) "foo" "bar") "foobar" "jit: concat") */
	(assert ((jit (lambda (s) (substr s 1 3))) "hello") "ell" "jit: substr")
	(assert ((jit (lambda (s) (substr s 2))) "hello") "llo" "jit: substr to end")
	/* JIT substr testbench: parity with interpreter across conversions and edges */
	(define jit_substr2 (jit (lambda (s i) (substr s i))))
	(define jit_substr3 (jit (lambda (s i n) (substr s i n))))
	(define _jit_id (lambda (x) x))
	(assert ((jit (lambda () (substr "abcdef" 1 3)))) "bcd" "jit substr constant-fold 3-arg")
	(assert ((jit (lambda () (substr "abcdef" 2)))) "cdef" "jit substr constant-fold 2-arg")
	(assert (equal? (jit_substr2 "abcdef" 0) (substr "abcdef" 0)) true "jit substr2 parity start")
	(assert (equal? (jit_substr2 "abcdef" 1) (substr "abcdef" 1)) true "jit substr2 parity offset 1")
	(assert (equal? (jit_substr2 "abcdef" 5) (substr "abcdef" 5)) true "jit substr2 parity last char")
	(assert (equal? (jit_substr3 "abcdef" 0 3) (substr "abcdef" 0 3)) true "jit substr3 parity prefix")
	(assert (equal? (jit_substr3 "abcdef" 1 3) (substr "abcdef" 1 3)) true "jit substr3 parity middle")
	(assert (equal? (jit_substr3 "abcdef" 3 0) (substr "abcdef" 3 0)) true "jit substr3 parity zero len")
	(assert (equal? (jit_substr3 "abcdef" 3 3) (substr "abcdef" 3 3)) true "jit substr3 parity suffix")
	(assert (equal? (jit_substr3 "abcdef" (+ 1 1) (+ 1 2)) (substr "abcdef" (+ 1 1) (+ 1 2))) true "jit substr3 parity arithmetic idx/len")
	(assert (equal? (jit_substr2 (_jit_id 123456) 2) (substr (_jit_id 123456) 2)) true "jit substr2 parity String conversion from int")
	(assert (equal? (jit_substr3 (_jit_id 123456) 2 3) (substr (_jit_id 123456) 2 3)) true "jit substr3 parity String conversion from int")
	(assert (equal? (jit_substr2 "abcdef" 2.0) (substr "abcdef" 2.0)) true "jit substr2 parity ToInt conversion from float")
	(assert (equal? (jit_substr3 "abcdef" 1.0 3.0) (substr "abcdef" 1.0 3.0)) true "jit substr3 parity ToInt conversion from float")
	(assert (equal? (jit_substr3 "äöüxyz" 0 6) (substr "äöüxyz" 0 6)) true "jit substr3 parity utf8 byte slicing")
	(assert ((jit (lambda (s) (toUpper s))) "hello") "HELLO" "jit: toUpper")
	(assert ((jit (lambda (s) (toLower s))) "HELLO") "hello" "jit: toLower")

	/* Pattern matching (strlike) */
	(assert ((jit (lambda (s) (strlike s "hello"))) "hello") true "jit: strlike exact")
	(assert ((jit (lambda (s) (strlike s "hello"))) "world") false "jit: strlike no match")
	(assert ((jit (lambda (s) (strlike s "h%"))) "hello") true "jit: strlike prefix %")
	(assert ((jit (lambda (s) (strlike s "%o"))) "hello") true "jit: strlike suffix %")
	(assert ((jit (lambda (s) (strlike s "h_llo"))) "hello") true "jit: strlike single _")
	(assert ((jit (lambda (s) (strlike s "%ll%"))) "hello") true "jit: strlike infix")
	(assert ((jit (lambda (s p) (strlike s p))) "hello" "h%") true "jit: strlike dynamic pattern")

	/* Deeply nested arithmetic */
	(assert ((jit (lambda (x) (+ (* x x) (* 2 x) 1))) 3) 16 "jit: x²+2x+1 = (x+1)²")
	(assert ((jit (lambda (x) (+ (* x x) (* 2 x) 1))) 5) 36 "jit: 5²+10+1 = 36")
	(assert ((jit (lambda (a b) (* (+ a b) (- a b)))) 7 3) 40 "jit: (a+b)(a-b) = a²-b²")
	(assert ((jit (lambda (x) (* (+ x 1) (+ x 2)))) 3) 20 "jit: (x+1)(x+2)")
	(assert ((jit (lambda (x) (- (* 3 (* x x)) (* 2 x)))) 4) 40 "jit: 3x²-2x")
	(assert ((jit (lambda (a b c) (+ (* a a) (* b b) (* c c)))) 3 4 5) 50 "jit: a²+b²+c² = 50")
	(assert ((jit (lambda (a b c) (* (+ a b) (+ b c) (+ a c)))) 1 2 3) 60 "jit: (a+b)(b+c)(a+c)")
	(assert ((jit (lambda (x) (+ (* (* x x) x) (* x x) x 1))) 2) 15 "jit: x³+x²+x+1")

	/* Chained operations with constants */
	(assert ((jit (lambda (x) (+ (* x 10) (* x 5) (* x 1)))) 3) 48 "jit: 10x+5x+x = 16x")
	(assert ((jit (lambda (x) (* (+ x 1) (- x 1)))) 5) 24 "jit: (x+1)(x-1) = x²-1")
	(assert ((jit (lambda (a b) (+ (* 3 a) (* 4 b)))) 2 5) 26 "jit: 3a+4b")
	(assert ((jit (lambda (a b) (- (* a b) (+ a b)))) 6 4) 14 "jit: ab-(a+b)")

	/* Four parameters */
	(assert ((jit (lambda (a b c d) (+ (* a d) (* b c)))) 2 3 4 5) 22 "jit: ad+bc")
	(assert ((jit (lambda (a b c d) (- (* a b) (* c d)))) 5 4 3 2) 14 "jit: ab-cd")
	(assert ((jit (lambda (a b c d) (* (+ a b) (+ c d)))) 1 2 3 4) 21 "jit: (a+b)(c+d)")

	/* Conditional with nested arithmetic */
	(assert ((jit (lambda (x) (if (> (* x x) 10) (* x 2) (+ x 1)))) 4) 8 "jit: if x²>10 then 2x else x+1 (true)")
	(assert ((jit (lambda (x) (if (> (* x x) 10) (* x 2) (+ x 1)))) 2) 3 "jit: if x²>10 then 2x else x+1 (false)")
	(assert ((jit (lambda (a b) (if (> (+ a b) 10) (* a b) (+ a b)))) 7 5) 35 "jit: if a+b>10 then a*b else a+b (true)")
	(assert ((jit (lambda (a b) (if (> (+ a b) 10) (* a b) (+ a b)))) 3 4) 7 "jit: if a+b>10 then a*b else a+b (false)")

	/* Nested comparisons and boolean logic */
	(assert ((jit (lambda (x) (and (> (* x 2) 5) (< (* x 3) 20)))) 3) true "jit: 2x>5 and 3x<20")
	(assert ((jit (lambda (x) (and (> (* x 2) 5) (< (* x 3) 20)))) 7) false "jit: 2x>5 and 3x<20 (false)")
	(assert ((jit (lambda (x) (or (< x 0) (> (* x x) 100)))) 11) true "jit: x<0 or x²>100")

	/* Mixed types and nil handling */
	(assert (nil? ((jit (lambda (x) (+ x nil))) 5)) true "jit: + with nil returns nil")
	(assert (nil? ((jit (lambda (x) (* x nil))) 5)) true "jit: * with nil returns nil")
	(assert ((jit (lambda (x) (if (nil? x) 0 x))) nil) 0 "jit: nil? check true")
	(assert ((jit (lambda (x) (if (nil? x) 0 x))) 42) 42 "jit: nil? check false")

	/* JIT emitter: nil? */
	(assert ((jit (lambda (x) (nil? x))) nil) true "jit: nil? of nil")
	(assert ((jit (lambda (x) (nil? x))) 0) false "jit: nil? of 0")
	(assert ((jit (lambda (x) (nil? x))) false) false "jit: nil? of false")
	(assert ((jit (lambda (x) (nil? x))) "") false "jit: nil? of empty string")

	/* JIT emitter: number? */
	(assert ((jit (lambda (x) (number? x))) 42) true "jit: number? of int")
	(assert ((jit (lambda (x) (number? x))) 3.14) true "jit: number? of float")
	(assert ((jit (lambda (x) (number? x))) "hello") false "jit: number? of string")
	(assert ((jit (lambda (x) (number? x))) nil) false "jit: number? of nil")
	(assert ((jit (lambda (x) (number? x))) true) false "jit: number? of bool")

	/* JIT emitter: ! and not */
	(assert ((jit (lambda (x) (! x))) true) false "jit: ! true")
	(assert ((jit (lambda (x) (! x))) false) true "jit: ! false")
	(assert ((jit (lambda (x) (not x))) true) false "jit: not true")
	(assert ((jit (lambda (x) (not x))) false) true "jit: not false")

	/* JIT emitter: + constant folding */
	(assert ((jit (lambda () (+ 3 4)))) 7 "jit: + constant fold int")
	(assert ((jit (lambda () (+ 1.5 2.5)))) 4.0 "jit: + constant fold float")
	(assert (nil? ((jit (lambda () (+ 1 nil))))) true "jit: + constant fold nil")

	/* JIT emitter: - */
	(assert ((jit (lambda (a b) (- a b))) 10 3) 7 "jit: a - b int")
	(assert ((jit (lambda (a b) (- a b))) 10.0 3.0) 7.0 "jit: a - b float")
	(assert (nil? ((jit (lambda (x) (- x nil))) 5)) true "jit: - with nil")
	(assert ((jit (lambda () (- 10 3)))) 7 "jit: - constant fold")

	/* JIT emitter: * */
	(assert ((jit (lambda (a b) (* a b))) 3 4) 12 "jit: a * b int")
	(assert ((jit (lambda (a b) (* a b))) 2.5 4.0) 10.0 "jit: a * b float")
	(assert (nil? ((jit (lambda (x) (* x nil))) 5)) true "jit: * with nil arg")
	(assert ((jit (lambda () (* 6 7)))) 42 "jit: * constant fold")

	/* JIT emitter: / */
	(assert ((jit (lambda (a b) (/ a b))) 10 4) 2.5 "jit: a / b")
	(assert ((jit (lambda (a b) (/ a b))) 10.0 2.0) 5.0 "jit: a / b float")
	(assert (nil? ((jit (lambda (x) (/ x nil))) 5)) true "jit: / with nil")
	(assert ((jit (lambda () (/ 10 4)))) 2.5 "jit: / constant fold")

	/* JIT emitter: < <= > >= */
	(assert ((jit (lambda (a b) (< a b))) 3 5) true "jit: 3 < 5")
	(assert ((jit (lambda (a b) (< a b))) 5 3) false "jit: 5 < 3")
	(assert ((jit (lambda (a b) (<= a b))) 3 3) true "jit: 3 <= 3")
	(assert ((jit (lambda (a b) (<= a b))) 4 3) false "jit: 4 <= 3")
	(assert ((jit (lambda (a b) (> a b))) 5 3) true "jit: 5 > 3")
	(assert ((jit (lambda (a b) (> a b))) 3 5) false "jit: 3 > 5")
	(assert ((jit (lambda (a b) (>= a b))) 3 3) true "jit: 3 >= 3")
	(assert ((jit (lambda (a b) (>= a b))) 2 3) false "jit: 2 >= 3")
	/* Float comparisons */
	(assert ((jit (lambda (a b) (< a b))) 1.5 2.5) true "jit: 1.5 < 2.5")
	(assert ((jit (lambda (a b) (> a b))) 2.5 1.5) true "jit: 2.5 > 1.5")
	/* Constant fold comparisons */
	(assert ((jit (lambda () (< 3 5)))) true "jit: < constant fold true")
	(assert ((jit (lambda () (< 5 3)))) false "jit: < constant fold false")
	(assert ((jit (lambda () (>= 3 3)))) true "jit: >= constant fold equal")

	/* JIT emitter: equal? */
	(assert ((jit (lambda (a b) (equal? a b))) 5 5) true "jit: equal? int same")
	(assert ((jit (lambda (a b) (equal? a b))) 5 6) false "jit: equal? int diff")
	(assert ((jit (lambda (a b) (equal? a b))) 3.14 3.14) true "jit: equal? float same")
	(assert ((jit (lambda () (equal? 5 5)))) true "jit: equal? constant fold")

	/* JIT emitter: int? — constant fold (LocImm) */
	(assert ((jit (lambda () (int? nil)))) false "jit: int? const nil")
	(assert ((jit (lambda () (int? true)))) false "jit: int? const bool true")
	(assert ((jit (lambda () (int? false)))) false "jit: int? const bool false")
	(assert ((jit (lambda () (int? "hello")))) false "jit: int? const string")
	(assert ((jit (lambda () (int? 3.14)))) false "jit: int? const float")

	/* JIT emitter: int? — register path (LocRegPair) with all types */
	(define _jit_int? (jit (lambda (x) (int? x))))
	(assert (_jit_int? nil) false "jit: int? reg nil")
	(assert (_jit_int? true) false "jit: int? reg bool true")
	(assert (_jit_int? false) false "jit: int? reg bool false")
	(assert (_jit_int? 3.14) false "jit: int? reg float")
	(assert (_jit_int? "hello") false "jit: int? reg string")
	(assert (_jit_int? (size "abc")) true "jit: int? reg int from size")
	(assert (_jit_int? (* 2 3)) true "jit: int? reg int from mul")
	(assert (_jit_int? (+ (size "abc") (size "de"))) true "jit: int? reg int from add")
	(assert (_jit_int? (* 0 1)) true "jit: int? reg int zero")
	(assert (_jit_int? (* -1 42)) true "jit: int? reg negative int")

	/* JIT emitter: int? — result feeds into other operations */
	(assert ((jit (lambda (x) (! (int? x)))) 3.14) true "jit: int? chained with !")
	(assert ((jit (lambda (x) (! (int? x)))) (size "a")) false "jit: int? chained with ! on int")

	/* JIT emitter: nil? — constant fold */
	(assert ((jit (lambda () (nil? nil)))) true "jit: nil? const nil")
	(assert ((jit (lambda () (nil? true)))) false "jit: nil? const bool")
	(assert ((jit (lambda () (nil? 3.14)))) false "jit: nil? const float")
	(assert ((jit (lambda () (nil? "hi")))) false "jit: nil? const string")

	/* JIT emitter: nil? — register path */
	(define _jit_nil? (jit (lambda (x) (nil? x))))
	(assert (_jit_nil? nil) true "jit: nil? reg nil")
	(assert (_jit_nil? true) false "jit: nil? reg bool true")
	(assert (_jit_nil? false) false "jit: nil? reg bool false")
	(assert (_jit_nil? 3.14) false "jit: nil? reg float")
	(assert (_jit_nil? "hello") false "jit: nil? reg string")
	(assert (_jit_nil? (size "abc")) false "jit: nil? reg int")
	(assert (_jit_nil? (* 2 3)) false "jit: nil? reg int from mul")

	/* number? with If/Phi/Jump */
	(define _jit_number? (jit (lambda (x) (number? x))))
	(assert (_jit_number? 42) true "jit: number? int")
	(assert (_jit_number? (size "abc")) true "jit: number? int from size")
	(assert (_jit_number? 3.14) true "jit: number? float")
	(assert (_jit_number? nil) false "jit: number? nil")
	(assert (_jit_number? true) false "jit: number? bool")
	(assert (_jit_number? "hello") false "jit: number? string")
	(assert (_jit_number? false) false "jit: number? bool false")

	/* nested JIT operator calls — test constant folding and type propagation */
	(define _jit_int_of_number (jit (lambda (x) (int? (number? x)))))
	(assert (_jit_int_of_number 42) false "jit: int?(number? 42) = false (bool)")
	(assert (_jit_int_of_number "hi") false "jit: int?(number? str) = false (bool)")

	(define _jit_nil_of_nil (jit (lambda (x) (nil? (nil? x)))))
	(assert (_jit_nil_of_nil nil) false "jit: nil?(nil? nil) = false (bool, not nil)")
	(assert (_jit_nil_of_nil 42) false "jit: nil?(nil? 42) = false (bool)")

	(define _jit_number_of_number (jit (lambda (x) (number? (number? x)))))
	(assert (_jit_number_of_number 42) false "jit: number?(number? 42) = false (bool)")

	/* critical: number? is multi-block, returns LocRegPair — Type must not be 0 */
	(define _jit_nil_of_number (jit (lambda (x) (nil? (number? x)))))
	(assert (_jit_nil_of_number 42) false "jit: nil?(number? 42) = false (bool, not nil)")
	(assert (_jit_nil_of_number "hi") false "jit: nil?(number? str) = false (bool, not nil)")

	(define _jit_int_of_nil (jit (lambda (x) (int? (nil? x)))))
	(assert (_jit_int_of_nil nil) false "jit: int?(nil? nil) = false (bool)")
	(assert (_jit_int_of_nil 5) false "jit: int?(nil? 5) = false (bool)")

	/* JIT arithmetic: + - with phi loop */
	(define _jit_add2 (jit (lambda (a b) (+ a b))))
	(assert (_jit_add2 3 4) 7 "jit: 3+4 = 7")
	(assert (_jit_add2 -1 1) 0 "jit: -1+1 = 0")
	(assert (_jit_add2 100 200) 300 "jit: 100+200 = 300")
	(define _jit_add3 (jit (lambda (a b c) (+ a b c))))
	(assert (_jit_add3 1 2 3) 6 "jit: 1+2+3 = 6")
	(define _jit_sub2 (jit (lambda (a b) (- a b))))
	(assert (_jit_sub2 10 3) 7 "jit: 10-3 = 7")
	(assert (_jit_sub2 0 5) -5 "jit: 0-5 = -5")
	/* mixed types: int+float promotes to float, nil propagates */
	(define _jit_add_if (jit (lambda (a b) (+ a b))))
	(assert (_jit_add_if 1 2.5) 3.5 "jit: 1+2.5 = 3.5 (int+float)")
	(assert (_jit_add_if 2.5 3.5) 6.0 "jit: 2.5+3.5 = 6.0 (float+float)")
	(define _jit_add3m (jit (lambda (a b c) (+ a b c))))
	(assert (_jit_add3m 1 2 3.0) 6.0 "jit: 1+2+3.0 = 6.0 (int+int+float)")
	(assert (_jit_add3m 1.0 2 3) 6.0 "jit: 1.0+2+3 = 6.0 (float+int+int)")
	(assert (nil? (_jit_add3m 1 nil 3)) true "jit: 1+nil+3 = nil")
	(define _jit_add4m (jit (lambda (a b c d) (+ a b c d))))
	(assert (_jit_add4m 1 2 3 4) 10 "jit: 1+2+3+4 = 10 (all int)")
	(assert (_jit_add4m 1 2 3.0 4) 10.0 "jit: 1+2+3.0+4 = 10.0 (mixed)")
	(assert (nil? (_jit_add4m 1 2 nil 4)) true "jit: 1+2+nil+4 = nil")
	/* nested: x*2+2 pattern */
	(define _jit_x2p2 (jit (lambda (x) (+ (* x 2) 2))))
	(assert (_jit_x2p2 5) 12 "jit: 5*2+2 = 12")
	(assert (_jit_x2p2 0) 2 "jit: 0*2+2 = 2")
	(assert (_jit_x2p2 -3) -4 "jit: -3*2+2 = -4")

	/* JIT panic propagation: Go(try/recover) → JIT → Go(error) → panic */
	(print "testing JIT panic propagation ...")
	(define jit_will_panic (jit (lambda (x) (error "jit-boom"))))
	(jit-warn-if-fallback jit_will_panic "jit: panic callback")
	(define jit_panic_result (try (lambda () (jit_will_panic 42)) (lambda (e) "caught")))
	(assert jit_panic_result "caught" "panic through JIT frame must be recoverable")

	/* JIT list callbacks: constant code shape with runtime values/captures */
	(define _jit_map_constant_callback (jit (lambda (values)
		(map values (lambda (_) 7)))))
	(jit-warn-if-fallback _jit_map_constant_callback "jit: map constant callback")
	(assert (equal? (_jit_map_constant_callback '(1 2 3)) '(7 7 7)) true "jit: map constant callback result")

	(define _jit_map_captured_callback (jit (lambda (values offset)
		(map values (lambda (value) (+ value offset))))))
	(jit-warn-if-fallback _jit_map_captured_callback "jit: map captured callback")
	(assert (equal? (_jit_map_captured_callback '(1 2 3) 10) '(11 12 13)) true "jit: map callback reads runtime capture")

	(define _jit_map_mut_callback (jit (lambda (values)
		(map_mut values (lambda (value) (+ value 1))))))
	(jit-warn-if-fallback _jit_map_mut_callback "jit: map_mut callback")
	(assert (equal? (_jit_map_mut_callback (list 3 4 5)) '(4 5 6)) true "jit: map_mut callback result")

	(define _jit_reduce_callback (jit (lambda (values neutral offset)
		(reduce values (lambda (acc value) (+ acc (+ value offset))) neutral))))
	(jit-warn-if-fallback _jit_reduce_callback "jit: reduce callback")
	(assert (_jit_reduce_callback '(1 2 3) 0 10) 36 "jit: reduce callback reads accumulator and runtime capture")

	(define _jit_map_dynamic_callback (jit (lambda (values callback)
		(map values callback))))
	(jit-warn-if-fallback _jit_map_dynamic_callback "jit: map dynamic callback")
	(assert (equal? (_jit_map_dynamic_callback '(2 3 4) (lambda (value) (* value 3))) '(6 9 12)) true "jit: map runtime callback trampoline")

	/* JIT match: compile-time pattern pruning and direct list destructuring */
	(define _jit_match_source (jit (lambda (src)
		(match src
			'((symbol alias) schema relation outer join) (list schema relation outer join)
			_ nil))))
	(jit-warn-if-fallback _jit_match_source "jit: fixed-list match")
	(assert (equal? (_jit_match_source (list (symbol "alias") "sales" "orders" false nil)) '("sales" "orders" false nil)) true "jit: fixed-list match binds fields")
	(assert (nil? (_jit_match_source (list (symbol "other") "sales" "orders" false nil))) true "jit: fixed-list literal mismatch falls through")
	(assert (nil? (_jit_match_source (list (symbol "alias") "sales" "orders"))) true "jit: fixed-list length mismatch falls through")

	(define _jit_match_cons (jit (lambda (values)
		(match values
			(cons head tail) (list head tail)
			_ nil))))
	(jit-warn-if-fallback _jit_match_cons "jit: cons match")
	(assert (equal? (_jit_match_cons '(1 2 3)) (list 1 (list 2 3))) true "jit: cons match binds head and tail")
	(assert (nil? (_jit_match_cons '())) true "jit: empty list rejects cons pattern")
	(assert (nth (_jit_match_cons fd2) 0) "k0" "jit: cons match normalizes FastDict head")
	(assert (count (nth (_jit_match_cons fd2) 1)) 19 "jit: cons match normalizes FastDict tail")

	(define _jit_match_typed (jit (lambda (value)
		(match value
			(number? number) (+ number 1)
			(string? text) text
			_ nil))))
	(jit-warn-if-fallback _jit_match_typed "jit: typed alternatives")
	(assert (_jit_match_typed 41) 42 "jit: number pattern selects numeric branch")
	(assert (_jit_match_typed "text") "text" "jit: string pattern selects string branch")
	(assert (nil? (_jit_match_typed false)) true "jit: typed patterns retain fallback")

	(define _jit_match_list_type (jit (lambda (value)
		(match value
			(list? items) items
			_ nil))))
	(jit-warn-if-fallback _jit_match_list_type "jit: list pattern")
	(assert (count (_jit_match_list_type fd2)) 20 "jit: list? pattern normalizes FastDict")
	(assert (nil? (_jit_match_list_type "not a list")) true "jit: list? pattern rejects scalar")

	(define _jit_match_eval (jit (lambda (value expected)
		(match value
			(eval expected) true
			_ false))))
	(jit-warn-if-fallback _jit_match_eval "jit: eval pattern")
	(assert (_jit_match_eval "same" "same") true "jit: eval pattern accepts runtime equality")
	(assert (_jit_match_eval "left" "right") false "jit: eval pattern rejects runtime inequality")

	(define _jit_match_known_list (jit (lambda (value)
		(match (list 'alias value)
			'((symbol alias) result) result
			'((symbol other) impossible) impossible
			nil))))
	(jit-warn-if-fallback _jit_match_known_list "jit: known-list match")
	(assert (_jit_match_known_list 17) 17 "jit: known list type and length are matched without fallback")

	/* alu.go: sql_abs */
	(print "testing sql_abs ...")
	(assert (equal? (sql_abs -5) 5) true "sql_abs of -5 = 5")
	(assert (equal? (sql_abs 3) 3) true "sql_abs of 3 = 3")
	(assert (equal? (sql_abs 0) 0) true "sql_abs of 0 = 0")
	(assert (nil? (sql_abs nil)) true "sql_abs of nil = nil")
	(assert (equal? (sql_abs -3.7) 3.7) true "sql_abs of -3.7 = 3.7")

	/* alu.go: sqrt */
	(print "testing sqrt ...")
	(assert (equal? (sqrt 4) 2.0) true "sqrt of 4 = 2.0")
	(assert (equal? (sqrt 0) 0.0) true "sqrt of 0 = 0.0")
	(assert (equal? (sqrt 1) 1.0) true "sqrt of 1 = 1.0")
	(assert (equal? (sqrt 2.25) 1.5) true "sqrt of 2.25 = 1.5")
	(assert (nil? (sqrt nil)) true "sqrt of nil = nil")
	(assert (nil? (sqrt -1)) true "sqrt of negative = nil")
	(assert (equal? (round (* 1000 (sqrt 2))) 1414.0) true "sqrt of 2 ~ 1.414")

	/* alu.go: equal_collate / notequal_collate */
	(print "testing collation equality ...")
	(assert (equal_collate "hello" "HELLO" "utf8_general_ci") true "equal_collate ci: hello=HELLO")
	(assert (equal_collate "hello" "HELLO" "utf8_bin") false "equal_collate bin: hello!=HELLO")
	(assert (equal_collate "hello" "hello" "utf8_bin") true "equal_collate bin: hello=hello")
	(assert (nil? (equal_collate nil "x" "utf8_bin")) true "equal_collate nil returns nil")
	(assert (notequal_collate "hello" "HELLO" "utf8_bin") true "notequal_collate bin: hello!=HELLO")
	(assert (notequal_collate "hello" "HELLO" "utf8_general_ci") false "notequal_collate ci: hello=HELLO -> false")
	(assert (nil? (notequal_collate nil "x" "utf8_bin")) true "notequal_collate nil returns nil")

	/* date.go: full coverage */
	(print "testing date functions ...")
	/* helper: identity function returning "any" type to bypass static int validation */
	(define _i (lambda (x) x))
	/* current_date returns a number <= now */
	(assert (number? (current_date)) true "current_date returns number")
	(assert (<= (current_date) (now)) true "current_date <= now")
	/* parse_date */
	(define dt1 (parse_date "2024-06-15"))
	(assert (number? dt1) true "parse_date returns number")
	(assert (nil? (parse_date nil)) true "parse_date nil returns nil")
	/* format_date */
	(assert (equal? (format_date dt1 "%Y-%m-%d") "2024-06-15") true "format_date Y-m-d")
	(assert (equal? (format_date dt1 "%Y") "2024") true "format_date year only")
	(assert (nil? (format_date nil "%Y")) true "format_date nil returns nil")
	/* extract_date */
	(assert (equal? (extract_date dt1 "YEAR") 2024) true "extract_date YEAR")
	(assert (equal? (extract_date dt1 "MONTH") 6) true "extract_date MONTH")
	(assert (equal? (extract_date dt1 "DAY") 15) true "extract_date DAY")
	(assert (nil? (extract_date nil "YEAR")) true "extract_date nil returns nil")
	/* date_add / date_sub (use _i to bypass static type validator for int param) */
	(define dt2 (date_add dt1 (_i 1) "DAY"))
	(assert (equal? (format_date dt2 "%Y-%m-%d") "2024-06-16") true "date_add 1 DAY")
	(define dt3 (date_add dt1 (_i 1) "MONTH"))
	(assert (equal? (format_date dt3 "%Y-%m-%d") "2024-07-15") true "date_add 1 MONTH")
	(define dt4 (date_add dt1 (_i 1) "YEAR"))
	(assert (equal? (format_date dt4 "%Y-%m-%d") "2025-06-15") true "date_add 1 YEAR")
	(define dt5 (date_sub dt1 (_i 1) "DAY"))
	(assert (equal? (format_date dt5 "%Y-%m-%d") "2024-06-14") true "date_sub 1 DAY")
	(assert (nil? (date_add nil (_i 1) "DAY")) true "date_add nil returns nil")
	(assert (nil? (date_sub nil (_i 1) "DAY")) true "date_sub nil returns nil")
	/* date_add units: HOUR, MINUTE, SECOND, WEEK */
	(define dt_h (date_add dt1 (_i 2) "HOUR"))
	(assert (equal? (format_date dt_h "%H") "02") true "date_add 2 HOUR")
	(define dt_m (date_add dt1 (_i 30) "MINUTE"))
	(assert (equal? (format_date dt_m "%i") "30") true "date_add 30 MINUTE")
	(define dt_s (date_add dt1 (_i 45) "SECOND"))
	(assert (equal? (format_date dt_s "%s") "45") true "date_add 45 SECOND")
	(define dt_w (date_add dt1 (_i 1) "WEEK"))
	(assert (equal? (format_date dt_w "%Y-%m-%d") "2024-06-22") true "date_add 1 WEEK")
	/* date_sub units: HOUR, MINUTE, SECOND, WEEK */
	(define dt_sh (date_sub (date_add dt1 (_i 5) "HOUR") (_i 2) "HOUR"))
	(assert (equal? (format_date dt_sh "%H") "03") true "date_sub 2 HOUR")
	/* extract_date: HOUR, MINUTE, SECOND */
	(define dt6 (parse_date "2024-06-15 14:30:45"))
	(assert (equal? (extract_date dt6 "HOUR") 14) true "extract_date HOUR")
	(assert (equal? (extract_date dt6 "MINUTE") 30) true "extract_date MINUTE")
	(assert (equal? (extract_date dt6 "SECOND") 45) true "extract_date SECOND")
	/* date_trunc_day */
	(assert (equal? (format_date (date_trunc_day dt6) "%H:%i:%s") "00:00:00") true "date_trunc_day zeroes time")
	(assert (nil? (date_trunc_day nil)) true "date_trunc_day nil returns nil")
	/* str_to_date */
	(define dt7 (str_to_date "15/06/2024" "%d/%m/%Y"))
	(assert (equal? (format_date dt7 "%Y-%m-%d") "2024-06-15") true "str_to_date custom format")
	(assert (nil? (str_to_date nil "%Y-%m-%d")) true "str_to_date nil returns nil")
	(assert (nil? (str_to_date "invalid" "%Y-%m-%d")) true "str_to_date invalid returns nil")
	/* date_sub MINUTE, SECOND, WEEK */
	(define dt_sm (date_sub (date_add dt1 (_i 30) "MINUTE") (_i 15) "MINUTE"))
	(assert (equal? (format_date dt_sm "%i") "15") true "date_sub MINUTE")
	(define dt_ss (date_sub (date_add dt1 (_i 45) "SECOND") (_i 20) "SECOND"))
	(assert (equal? (format_date dt_ss "%s") "25") true "date_sub SECOND")
	(define dt_sw (date_sub (date_add dt1 (_i 2) "WEEK") (_i 1) "WEEK"))
	(assert (equal? (format_date dt_sw "%Y-%m-%d") "2024-06-22") true "date_sub WEEK")
	/* format_date with float timestamp (toTime tagFloat) - use _i to bypass type check */
	(assert (equal? (format_date (_i 0.0) "%Y") "1970") true "format_date float timestamp epoch")
	/* format_date with invalid string returns nil (toTime tagString fail) */
	(assert (nil? (format_date (_i "not-a-date") "%Y")) true "format_date bad string returns nil")
	/* format_date %% literal percent */
	(assert (equal? (format_date dt1 "100%%") "100%") true "format_date %% literal percent")
	/* str_to_date with time components %H %i %s */
	(define dt_hms (str_to_date "14:30:45" "%H:%i:%s"))
	(assert (equal? (format_date dt_hms "%H:%i:%s") "14:30:45") true "str_to_date time %H:%i:%s")
	/* parse_date on already-parsed date (tagDate passthrough) */
	(assert (equal? (parse_date dt1) dt1) true "parse_date on date is identity")
	/* parse_date on integer (tagInt) - use _i to bypass static type check */
	(assert (number? (parse_date (_i 1718451045))) true "parse_date on int returns date")

	/* list.go: produce */
	(print "testing produce ...")
	(assert (equal? (produce 0 (lambda (x) (< x 5)) (lambda (x) (+ x 1))) '(0 1 2 3 4)) true "produce 0..4")
	(assert (equal? (produce 10 (lambda (x) false) (lambda (x) x)) '()) true "produce empty on false init")
	/* list.go: edge cases */
	(assert (equal? (cdr '()) '()) true "cdr on empty list returns empty")
	(assert (equal? (reduce '(10 20 30) (lambda (acc x) (+ acc x)) 0) 60) true "reduce with neutral")
	(assert (nil? (reduce '() (lambda (acc x) (+ acc x)))) true "reduce empty list no neutral returns nil")
	(assert (equal? (merge '(1 2) '(3 4)) '(1 2 3 4)) true "merge multi-arg")
	(assert (equal? (merge_unique '(1 2) '(2 3)) '(1 2 3)) true "merge_unique multi-arg")
	(assert (equal? (find '(10 20 30) (lambda (x) (> x 15))) 20) true "find returns first matching element")
	(assert (equal? (find '(10 20 30) (lambda (x) (> x 100)) 99) 99) true "find default when missing")
	(assert (has_assoc? nil "key") false "has_assoc? on nil returns false")
	(assert (nil? (get_assoc nil "key")) true "get_assoc on nil returns nil")

	/* list.go: get_assoc */
	(print "testing get_assoc ...")
	(assert (equal? (get_assoc (list "a" 1 "b" 2) "a") 1) true "get_assoc finds key")
	(assert (nil? (get_assoc (list "a" 1) "z")) true "get_assoc missing key returns nil")
	(assert (equal? (get_assoc (list "a" 1) "z" 99) 99) true "get_assoc missing key returns default")
	/* get_assoc on FastDict */
	(define big_ga (reduce (produceN 20) (lambda (acc i) (set_assoc acc (concat "k" i) i)) '()))
	(assert (equal? (get_assoc big_ga "k5") 5) true "get_assoc on FastDict")
	(assert (equal? (get_assoc big_ga "missing" -1) -1) true "get_assoc FastDict missing key with default")

	/* strings.go: sql_substr (1-based) */
	(print "testing additional string functions ...")
	(assert (equal? (sql_substr "hello" 2 3) "ell") true "sql_substr 1-based with length")
	(assert (equal? (sql_substr "hello" 2) "ello") true "sql_substr 1-based to end")
	(assert (equal? (sql_substr "hello" 10) "") true "sql_substr out of bounds returns empty")
	(assert (nil? (sql_substr nil 1 3)) true "sql_substr nil returns nil")
	/* TODO: jit sql_substr tests crash — disabled until emitter is fixed
	(define jit_sql_substr2 (jit (lambda (s i) (sql_substr s i))))
	(define jit_sql_substr3 (jit (lambda (s i n) (sql_substr s i n))))
	(assert ((jit (lambda () (sql_substr "abcdef" 2 3)))) "bcd" "jit sql_substr constant-fold")
	(assert (equal? (jit_sql_substr2 "hello" 2) (sql_substr "hello" 2)) true "jit sql_substr2 parity basic")
	(assert (equal? (jit_sql_substr2 "hello" 0) (sql_substr "hello" 0)) true "jit sql_substr2 parity clamp start")
	(assert (equal? (jit_sql_substr3 "hello" 2 3) (sql_substr "hello" 2 3)) true "jit sql_substr3 parity basic")
	(assert (equal? (jit_sql_substr3 "hello" 4 10) (sql_substr "hello" 4 10)) true "jit sql_substr3 parity long len")
	(assert (equal? (jit_sql_substr3 "hello" 2 -1) (sql_substr "hello" 2 -1)) true "jit sql_substr3 parity negative len")
	(assert (equal? (jit_sql_substr3 "äöüxyz" 2 4) (sql_substr "äöüxyz" 2 4)) true "jit sql_substr3 parity utf8 byte slicing")
	(assert (nil? (jit_sql_substr3 nil 1 3)) true "jit sql_substr nil returns nil")
	*/

	/* strings.go: strlike_cs (case-sensitive) */
	(assert (strlike_cs "Hello" "H%") true "strlike_cs case-sensitive prefix match")
	(assert (strlike_cs "Hello" "h%") false "strlike_cs case-sensitive no match")
	(assert (strlike_cs "abc" "a_c") true "strlike_cs single char wildcard")

	/* strings.go: trim functions */
	(assert (equal? (strtrim "  hello  ") "hello") true "strtrim both ends")
	(assert (equal? (strltrim "  hello  ") "hello  ") true "strltrim left only")
	(assert (equal? (strrtrim "  hello  ") "  hello") true "strrtrim right only")
	(assert (equal? (sql_trim "  hello  ") "hello") true "sql_trim both ends")
	(assert (equal? (sql_ltrim "  hello  ") "hello  ") true "sql_ltrim left only")
	(assert (equal? (sql_rtrim "  hello  ") "  hello") true "sql_rtrim right only")
	(assert (nil? (sql_trim nil)) true "sql_trim nil returns nil")
	(assert (nil? (sql_ltrim nil)) true "sql_ltrim nil returns nil")
	(assert (nil? (sql_rtrim nil)) true "sql_rtrim nil returns nil")

	/* strings.go: string_repeat */
	(assert (equal? (string_repeat "ab" 3) "ababab") true "string_repeat 3x")
	(assert (equal? (string_repeat "x" 0) "") true "string_repeat 0 = empty")
	(assert (nil? (string_repeat nil 3)) true "string_repeat nil returns nil")
	/* strings.go: strlike with explicit collation */
	(assert (strlike "Hello" "h%" "utf8_general_ci") true "strlike explicit ci collation")
	(assert (strlike "Hello" "h%" "utf8_bin") false "strlike explicit bin collation is CS")
	/* strings.go: sql_substr edge cases */
	(assert (equal? (sql_substr "hello" 0 3) "hel") true "sql_substr start=0 clamped")
	(assert (equal? (sql_substr "hello" 4 10) "lo") true "sql_substr length exceeds remaining")
	/* strings.go: regexp_replace nil + direct */
	(assert (nil? (regexp_replace nil "foo" "bar")) true "regexp_replace nil returns nil")
	/* strings.go: collate with language aliases */
	(define less_en (collate "en"))
	(assert (less_en "a" "b") true "english collation a<b")
	(define less_rev (collate "en" true))
	(assert (less_rev "b" "a") true "english reverse b before a")

	/* strings.go: hash functions */
	(assert (equal? (fnv_hash "hello") (fnv_hash "hello")) true "fnv_hash is deterministic")
	(assert (equal? (fnv_hash "hello") (fnv_hash "world")) false "fnv_hash differs for different inputs")
	(assert (strlen (fnv_hash "test")) 16 "fnv_hash returns 16-char hex string")
	(assert (equal? (fnv_hash "") (fnv_hash "")) true "fnv_hash of empty string is deterministic")
	(define structural_hash_tree (list (quote call) (list (quote +) 1 2) "x"))
	(assert (equal? (stable_structural_hash structural_hash_tree false)
		(fnv_hash (string structural_hash_tree))) true "stable structural string hash preserves generated names")
	(assert (equal? (stable_structural_hash structural_hash_tree true)
		(fnv_hash (serialize structural_hash_tree))) true "stable structural serialize hash preserves generated names")
	(assert (equal? (stable_structural_hash (source "hash-test" 1 1 structural_hash_tree) true)
		(fnv_hash (serialize structural_hash_tree))) true "stable structural serialize hash unwraps SourceInfo")
	(define optimized_string_hash (serialize (optimize (list
		(quote lambda) (list (quote value))
		(list (quote fnv_hash) (list (quote string) (quote value)))))))
	(assert (strlike optimized_string_hash "%stable_structural_hash%false%") true
		"fnv_hash optimizer fuses the builtin string producer")
	(define optimized_serialize_hash (serialize (optimize (list
		(quote lambda) (list (quote value))
		(list (quote fnv_hash) (list (quote serialize) (quote value)))))))
	(assert (strlike optimized_serialize_hash "%stable_structural_hash%true%") true
		"fnv_hash optimizer fuses the builtin serializer producer")
	(define optimized_plain_hash (serialize (optimize (list
		(quote lambda) (list (quote value))
		(list (quote fnv_hash) (quote value))))))
	(assert (strlike optimized_plain_hash "%fnv_hash%") true
		"fnv_hash optimizer preserves non-producer calls")
	(define optimized_hash_payload (serialize (optimize (list
		(quote lambda) (list (quote left) (quote right))
		(list (quote stable_structural_hash)
			(list (quote list) (quote left) (quote right)) true)))))
	(assert (strlike optimized_hash_payload "%!list%") true
		"structural hash payload is stack-backed because it cannot escape")

	/* Compile-local structural catalogs keep small collections linear, promote
	at the fifth distinct key, resolve collisions with equal?, and freeze before
	parallel reads. */
	(define structural_small (make_structural_catalog))
	(structural_small 42 "integer")
	(structural_small 42.0 "numeric replacement")
	(structural_small (symbol "named") "symbol")
	(structural_small (list (quote nested) (list (quote shared) (quote leaf))) "tree")
	(structural_small (source "catalog-test" 2 3 '(wrapped key)) "source")
	(structural_small (parse_date "2024-06-15") "date")
	(structural_small "2024-06-15" "date replacement")
	(assert (structural_small (list (quote nested) (list (quote shared) (quote leaf)))) "tree" "structural catalog supports collection-time lookup")
	(assert (nil? (structural_small '(not present))) true "structural catalog collection-time lookup reports misses")
	(define structural_small_frozen (structural_small))
	(assert (structural_small_frozen 42) "numeric replacement" "structural catalog honors int/float equality")
	(assert (structural_small_frozen "named") "symbol" "structural catalog honors string/symbol equality")
	(assert (structural_small_frozen (list (quote nested) (list (quote shared) (quote leaf)))) "tree" "structural catalog finds equal nested lists")
	(assert (structural_small_frozen '(wrapped key)) "source" "structural catalog unwraps SourceInfo")
	(assert (structural_small_frozen (parse_date "2024-06-15")) "date replacement" "structural catalog honors date/string equality")
	(assert (nil? (structural_small_frozen (list (quote nested) (list (quote shared) (quote other))))) true "structural catalog keeps unequal trees distinct")
	(define structural_four (make_structural_catalog))
	(reduce (produceN 4) (lambda (_ i) (structural_four (list (quote boundary) i) i)) nil)
	(define structural_four_frozen (structural_four))
	(assert (structural_four_frozen (list (quote boundary) 3)) 3 "four-key structural catalog stays readable below promotion")
	(define structural_five (make_structural_catalog))
	(reduce (produceN 5) (lambda (_ i) (structural_five (list (quote boundary) i) i)) nil)
	(structural_five (list (quote boundary) 2) "duplicate condition")
	(define structural_five_frozen (structural_five))
	(assert (structural_five_frozen (list (quote boundary) 2)) "duplicate condition" "fifth key promotes and duplicate conditions replace atomically")
	(define structural_collision (make_structural_catalog true))
	(reduce (produceN 8) (lambda (_ i) (structural_collision (concat "collision" i) i)) nil)
	(define structural_collision_frozen (structural_collision))
	(assert (structural_collision_frozen "collision6") 6 "structural catalog resolves deliberate collisions with equality")
	(assert (nil? (structural_collision_frozen "collision99")) true "structural collision bucket cannot create false identity")
	(define structural_literal_ast (list (quote if) true))
	(define structural_probe_ast (list (quote if) (list (quote probe))))
	(define structural_ast_catalog (make_structural_catalog))
	(structural_ast_catalog structural_literal_ast "literal")
	(structural_ast_catalog structural_probe_ast "probe")
	(define structural_ast_frozen (structural_ast_catalog))
	(assert (structural_ast_frozen structural_literal_ast) "literal"
		"structural catalog keeps literal truth separate from a truthy AST")
	(assert (structural_ast_frozen structural_probe_ast) "probe"
		"structural catalog keeps a truthy AST separate from literal truth")
	(define structural_ast_index (make_structural_index
		(list structural_literal_ast structural_probe_ast)
		(list structural_literal_ast structural_probe_ast)))
	(assert (structural_ast_index structural_literal_ast) 0
		"structural index preserves the literal AST slot")
	(assert (structural_ast_index structural_probe_ast) 1
		"structural index preserves the probe AST slot")
	(define structural_concurrent (make_structural_catalog))
	(parallelN 64 (lambda (i) (structural_concurrent (list (quote concurrent) i) i)))
	(define structural_concurrent_frozen (structural_concurrent))
	(assert (reduce (produceN 64) (lambda (ok i)
		(and ok (equal? (structural_concurrent_frozen (list (quote concurrent) i)) i))) true)
		true "atomic structural catalog publication loses no concurrent entries")
	(define structural_mutable_root (reduce (produceN 8)
		(lambda (acc i) (set_assoc acc (concat "mutable" i) i)) '()))
	(assert (try
		(lambda () (begin (define invalid_catalog (make_structural_catalog))
			(invalid_catalog structural_mutable_root true) false))
		(lambda (_error) true)) true "structural catalog rejects mutable FastDict roots")
	(assert (equal? (sha1 "hello") (sha1 "hello")) true "sha1 is deterministic")
	(assert (equal? (sha1 "hello") (sha1 "world")) false "sha1 differs for different inputs")
	(assert (strlen (sha1 "test")) 40 "sha1 returns 40-char hex string")
	(assert (equal? (sha256 "hello") (sha256 "hello")) true "sha256 is deterministic")
	(assert (equal? (sha256 "hello") (sha256 "world")) false "sha256 differs for different inputs")
	(assert (strlen (sha256 "test")) 64 "sha256 returns 64-char hex string")

	/* scm.go: string (type conversion) / printer.go coverage */
	(print "testing type conversion and apply ...")
	(assert (equal? (string 42) "42") true "string converts number to string")
	(assert (equal? (string "abc") "abc") true "string on string is identity")
	(assert (equal? (string nil) "nil") true "string of nil")
	(assert (equal? (string true) "true") true "string of true")
	(assert (equal? (string false) "false") true "string of false")
	(assert (equal? (string 3.14) "3.14") true "string of float")
	(assert (equal? (string +) "[native func]") true "string of native func")
	/* serialize coverage (printer.go:SerializeEx) - use _i to bypass type checks */
	(assert (equal? (serialize (_i true)) "true") true "serialize true")
	(assert (equal? (serialize (_i false)) "false") true "serialize false")
	(assert (equal? (serialize (_i nil)) "nil") true "serialize nil")
	(assert (equal? (serialize (_i 42)) "42") true "serialize int")
	(assert (equal? (serialize (_i 3.14)) "3.14") true "serialize float")
	(assert (equal? (serialize (_i "hello")) "\"hello\"") true "serialize string")
	/* serialize: symbol */
	(assert (equal? (serialize (_i 'foo)) "foo") true "serialize symbol")
	/* serialize: list */
	(assert (equal? (serialize '(1 2 3)) "(1 2 3)") true "serialize list")
	(assert (equal? (serialize '()) "()") true "serialize empty list")
	(assert (equal? (serialize '('+ '('* 2 3) 4)) "(+ (* 2 3) 4)") true "serialize nested list")
	/* serialize: lambda code template */
	(assert (equal? (serialize '('lambda '('x) '('+ 'x 1))) "(lambda (x) (+ x 1))") true "serialize lambda template")
	/* serialize: list of symbols */
	(assert (equal? (serialize '('a 'b 'c)) "(a b c)") true "serialize list of symbols")
	/* serialize: string with escapes (backslash, quotes, newline) */
	(assert (equal? (serialize (_i "he said \"hi\"")) "\"he said \\\"hi\\\"\"") true "serialize string with embedded quotes")
	(assert (equal? (serialize (_i "line1\nline2")) "\"line1\\nline2\"") true "serialize string with newline")
	/* serialize: negative number */
	(assert (equal? (serialize (_i -42)) "-42") true "serialize negative int")
	/* serialize: tagProc - real lambda (not a quoted code template) */
	(assert (> (strlen (serialize (_i (lambda (x y) (+ x y))))) 5) true "serialize real lambda (tagProc)")
	/* serialize: tagFastDict via set_assoc */
	(define ser_fd (set_assoc (set_assoc (list) "a" 1) "b" 2))
	(assert (> (strlen (serialize (_i ser_fd))) 3) true "serialize FastDict (tagFastDict)")
	/* serialize: tagFunc - native function */
	(assert (equal? (serialize (_i +)) "+") true "serialize native func +")
	(assert (equal? (serialize (_i concat)) "concat") true "serialize native func concat")
	/* serialize: symbol with special chars (unquote branch) */
	(assert (equal? (serialize (_i (symbol "hello world"))) "(unquote \"hello world\")") true "serialize symbol with space triggers unquote")
	(assert (equal? (serialize (_i (symbol "a\"b"))) "(unquote \"a\\\"b\")") true "serialize symbol with quote triggers unquote")
	/* serialize: (outer ...) pattern */
	(assert (equal? (serialize (list (symbol "outer") (symbol "x"))) "(outer x)") true "serialize outer expression")
	/* serialize: list starting with 'list symbol (quote shorthand '(...) ) */
	(assert (strlike (serialize (list (symbol "list") 1 2 3)) "%1 2 3%") true "serialize list-prefixed list")
	/* serialize: collate function (serializeNativeFunc collate branch) */
	(define ser_coll (collate "en"))
	(assert (equal? (serialize (_i ser_coll)) "(collate \"en\" false)") true "serialize collate function")
	(define ser_coll_rev (collate "en" true))
	(assert (equal? (serialize (_i ser_coll_rev)) "(collate \"en\" true)") true "serialize reverse collate function")
	/* serialize: optimized lambda with NumVars (serializeProcShallow NumVars > 0 branch) */
	(define ser_opt_lam (eval (optimize '('lambda '('a 'b) '('+ 'a 'b)))))
	(assert (> (strlen (serialize (_i ser_opt_lam))) 5) true "serialize optimized lambda with NumVars")
	/* serialize roundtrip: eval serialized code */
	(assert (equal? (eval (scheme (serialize '(+ 1 2)) "ser-test.scm")) 3) true "serialize roundtrip eval")
	(define scmer_json_roundtrip '('query-block "db" '(list) nil))
	(assert (equal? (json_decode_scmer (json_encode scmer_json_roundtrip)) scmer_json_roundtrip) true "Scmer JSON roundtrip preserves symbols and lists")

	/* String() coverage (printer.go:String) */
	(assert (equal? (string '()) "()") true "string of empty list")
	(assert (equal? (string '(1 2 3)) "(1 2 3)") true "string of list")
	(assert (equal? (string 'hello) "hello") true "string of symbol")
	/* string of real lambda (tagProc) */
	(assert (> (strlen (string (lambda (a b) (* a b)))) 5) true "string of real lambda (tagProc)")
	/* string of FastDict */
	(define str_fd (set_assoc (set_assoc (list) "a" 1) "b" 2))
	(assert (> (strlen (string str_fd)) 3) true "string of FastDict")

	/* scm.go: ApplyEx coverage */
	(print "testing ApplyEx coverage ...")
	/* assoc list: even-length, missing key returns nil */
	(define assoc_even (list "a" 1 "b" 2))
	(assert (equal? (assoc_even "nonexistent") nil) true "assoc even-length missing key returns nil")
	/* assoc list: odd-length, missing key returns last element as default */
	(define assoc_odd (list "a" 1 "fallback"))
	(assert (equal? (assoc_odd "missing") "fallback") true "assoc odd-length default value")
	/* assoc list: key found returns value */
	(assert (equal? (assoc_even "a") 1) true "assoc even-length key found")
	(assert (equal? (assoc_even "b") 2) true "assoc even-length second key found")
	/* FastDict: missing key returns nil */
	(define fd_apply (reduce (produceN 20) (lambda (acc i) (set_assoc acc (concat "k" i) i)) '()))
	(assert (equal? (fd_apply "nonexistent_key") nil) true "FastDict missing key returns nil")
	/* FastDict: key found */
	(assert (equal? (fd_apply "k0") 0) true "FastDict key found k0")
	(assert (equal? (fd_apply "k19") 19) true "FastDict key found k19")

	/* scm.go: apply */
	(assert (equal? (apply + '(1 2 3)) 6) true "apply + to list")
	(assert (equal? (apply concat '("a" "b" "c")) "abc") true "apply concat to list")

	/* optimizer.go: OptimizeEx coverage */
	(print "testing OptimizeEx coverage ...")
	/* SourceInfo wrapping: parse code string, then optimize (triggers tagSourceInfo path) */
	/* (scheme "..." "file.scm") produces SourceInfo-wrapped AST; optimize must fold constants through it */
	(assert (equal? (optimize (scheme "(+ 1 2)" "opt-test.scm")) 3) true "optimize folds constant through SourceInfo")
	(assert (equal? (optimize (scheme "(+ 3 4)" "opt-test2.scm")) 7) true "optimize folds constant through SourceInfo 2")
	/* optimize: constant folding through nested SourceInfo (concat) */
	(assert (equal? (optimize (scheme "(concat \"a\" \"b\")" "opt-si-concat.scm")) "ab") true "optimize folds concat through SourceInfo")
	/* optimize: SourceInfo wrapping a non-constant expression (quoted lambda data) */
	(assert (strlike (serialize (_i (optimize '('lambda '('x) '('+ 'x 1))))) "%lambda%") true "optimize quoted lambda data preserves lambda")

	/* scm.go: error (via try) */
	(define err_caught (newsession))
	(try (lambda () (error "test error")) (lambda (e) (err_caught "msg" e)))
	(assert (strlike (err_caught "msg") "%test error%") true "error throws and try catches")

	/* scm.go: time */
	(define time_result (time (+ 2 3)))
	(assert (equal? time_result 5) true "time returns the computed value")

	/* scm.go: parallel */
	(define par_done (newsession))
	(par_done "a" false)
	(par_done "b" false)
	(parallel (par_done "a" true) (par_done "b" true))
	(assert (par_done "a") true "parallel executed branch a")
	(assert (par_done "b") true "parallel executed branch b")
	(assert (equal? (parallelN 6 (lambda (i) (+ i 10))) '(10 11 12 13 14 15)) true "parallelN returns stable index order")
	(assert (equal? (produceN_mut 4 (lambda (i) (* i 3)) '(nil nil nil nil)) '(0 3 6 9)) true "produceN_mut writes into target")
	(assert (equal? (parallelN_mut 4 (lambda (i) (+ i 1)) '(nil nil nil nil)) '(1 2 3 4)) true "parallelN_mut writes into target")
	(assert (equal? (produceN_mut 4 (lambda (i) (+ i 1)) nil) nil) true "produceN_mut nil-target avoids allocation")
	(assert (equal? (parallelN_mut 4 (lambda (i) (+ i 1)) nil) nil) true "parallelN_mut nil-target avoids allocation")

	/* list.go: parallel_map / parallel_map_mut */
	(assert (equal? (parallel_map '() (lambda (x) x)) '()) true "parallel_map empty list")
	(assert (equal? (parallel_map '(1) (lambda (x) (* x 10))) '(10)) true "parallel_map single element")
	(assert (equal? (parallel_map '(1 2 3 4 5) (lambda (x) (* x 2))) '(2 4 6 8 10)) true "parallel_map preserves order")
	(assert (equal? (parallel_map (produceN 100) (lambda (i) (* i i))) (map (produceN 100) (lambda (i) (* i i)))) true "parallel_map matches sequential map for 100 elements")
	(assert (equal? (parallel_map_mut '(10 20 30) (lambda (x) (+ x 1))) '(11 21 31)) true "parallel_map_mut basic")
	(assert (equal? (parallel_map_mut (produceN 50) (lambda (i) (+ i 100))) (map (produceN 50) (lambda (i) (+ i 100)))) true "parallel_map_mut matches sequential map for 50 elements")

	/* scm.go: for_mut (optimizer-internal but callable) */
	(assert (equal? (for_mut (list 0) (lambda (x) (< x 5)) (lambda (x) (list (+ x 1)))) '(5)) true "for_mut counts to 5")

	/* sync.go: numcpu / memstats */
	(print "testing system info ...")
	(assert (> (numcpu) 0) true "numcpu > 0")
	(define ms (memstats))
	(assert (> (ms "alloc") 0) true "memstats alloc > 0")
	(assert (> (ms "sys") 0) true "memstats sys > 0")
	(assert (has_assoc? ms "heap_alloc") true "memstats has heap_alloc")
	(assert (has_assoc? ms "heap_sys") true "memstats has heap_sys")
	(assert (has_assoc? ms "total_alloc") true "memstats has total_alloc")

	/* _mut variants (optimizer internal, but test directly for coverage) */
	(print "testing _mut variants ...")
	(assert (equal? (map_mut '(1 2 3) (lambda (x) (* x 2))) '(2 4 6)) true "map_mut doubles")
	(assert (equal? (mapIndex_mut '(10 20) (lambda (i v) (string i))) '("0" "1")) true "mapIndex_mut")
	(assert (equal? (filter_mut '(1 2 3 4) (lambda (x) (> x 2))) '(3 4)) true "filter_mut keeps >2")
	(assert (equal? (append_mut '(1 2) 3 4) '(1 2 3 4)) true "append_mut extends list")
	(assert (equal? (append_unique_mut '(1 2 2) 2 3) '(1 2 2 3)) true "append_unique_mut deduplicates")
	(define d_mut (list "a" 1))
	(set d_mut (set_assoc_mut d_mut "b" 2))
	(assert (equal? (get_assoc d_mut "b") 2) true "set_assoc_mut adds key")
	(define d_mut2 (merge_assoc_mut (list "x" 1) (list "y" 2)))
	(assert (equal? (get_assoc d_mut2 "y") 2) true "merge_assoc_mut merges")
	(define d_mut3 (map_assoc_mut (list "a" 1 "b" 2) (lambda (k v) (+ v 10))))
	(assert (equal? (get_assoc d_mut3 "a") 11) true "map_assoc_mut increments")
	(define d_mut3b (mapkey_assoc_mut (list "a" 1 "b" 2) (lambda (k v) (concat k k))))
	(assert (equal? (get_assoc d_mut3b "aa") 1) true "mapkey_assoc_mut remaps key")
	(define d_mut3c (mapkey_assoc_mut (list "a" 1 "b" 2) (lambda (k v) "same")))
	(assert (equal? (get_assoc d_mut3c "same") 2) true "mapkey_assoc_mut collision last wins")
	(define d_mut4 (filter_assoc_mut (list "a" 1 "b" 20) (lambda (k v) (> v 5))))
	(assert (has_assoc? d_mut4 "b") true "filter_assoc_mut keeps b")
	(assert (has_assoc? d_mut4 "a") false "filter_assoc_mut drops a")
	(define d_mut5 (extract_assoc_mut (list "a" 1 "b" 2) (lambda (k v) v)))
	(assert (equal? d_mut5 '(1 2)) true "extract_assoc_mut extracts values")

	/* window_mut / window_flush */
	(print "testing window_mut / window_flush ...")
	/* LAG(col1, 1): window_size=2, stride=1, skip=0 */
	(define _win_results (newsession))
	(_win_results "items" '())
	(define _win_lag1 (list 0 0 1 nil nil))
	(set _win_lag1 (window_mut _win_lag1 (lambda (oldest newest) (_win_results "items" (merge (_win_results "items") (list oldest)))) (list 10)))
	(assert (equal? (_win_results "items") '(nil)) true "window_mut LAG stride=1 row1 emits nil")
	(set _win_lag1 (window_mut _win_lag1 (lambda (oldest newest) (_win_results "items" (merge (_win_results "items") (list oldest)))) (list 20)))
	(assert (equal? (_win_results "items") '(nil 10)) true "window_mut LAG stride=1 row2 emits 10")
	(set _win_lag1 (window_mut _win_lag1 (lambda (oldest newest) (_win_results "items" (merge (_win_results "items") (list oldest)))) (list 30)))
	(assert (equal? (_win_results "items") '(nil 10 20)) true "window_mut LAG stride=1 row3 emits 20")

	/* LEAD(col1, 1): window_size=2, stride=1, skip=1 */
	(define _win_results2 (newsession))
	(_win_results2 "items" '())
	(define _win_lead1 (list 1 0 1 nil nil))
	(set _win_lead1 (window_mut _win_lead1 (lambda (oldest newest) (_win_results2 "items" (merge (_win_results2 "items") (list newest)))) (list 10)))
	(assert (equal? (_win_results2 "items") '()) true "window_mut LEAD skip=1 row1 no emit")
	(set _win_lead1 (window_mut _win_lead1 (lambda (oldest newest) (_win_results2 "items" (merge (_win_results2 "items") (list newest)))) (list 20)))
	(assert (equal? (_win_results2 "items") '(20)) true "window_mut LEAD row2 emits newest=20")
	(set _win_lead1 (window_mut _win_lead1 (lambda (oldest newest) (_win_results2 "items" (merge (_win_results2 "items") (list newest)))) (list 30)))
	(assert (equal? (_win_results2 "items") '(20 30)) true "window_mut LEAD row3 emits newest=30")
	/* flush remaining */
	(window_flush _win_lead1 (lambda (oldest newest) (_win_results2 "items" (merge (_win_results2 "items") (list newest)))) 1)
	(assert (equal? (_win_results2 "items") '(20 30 nil)) true "window_flush LEAD emits nil for last row")

	/* LEAD(col1, 2): window_size=3, stride=1, skip=2 */
	(define _win_results3 (newsession))
	(_win_results3 "items" '())
	(define _win_lead2 (list 2 0 1 nil nil nil))
	(set _win_lead2 (window_mut _win_lead2 (lambda (a b c) (_win_results3 "items" (merge (_win_results3 "items") (list c)))) (list 10)))
	(set _win_lead2 (window_mut _win_lead2 (lambda (a b c) (_win_results3 "items" (merge (_win_results3 "items") (list c)))) (list 20)))
	(assert (equal? (_win_results3 "items") '()) true "window_mut LEAD(2) skips first 2 rows")
	(set _win_lead2 (window_mut _win_lead2 (lambda (a b c) (_win_results3 "items" (merge (_win_results3 "items") (list c)))) (list 30)))
	(assert (equal? (_win_results3 "items") '(30)) true "window_mut LEAD(2) row3 emits newest=30")
	(set _win_lead2 (window_mut _win_lead2 (lambda (a b c) (_win_results3 "items" (merge (_win_results3 "items") (list c)))) (list 40)))
	(assert (equal? (_win_results3 "items") '(30 40)) true "window_mut LEAD(2) row4 emits 40")
	(window_flush _win_lead2 (lambda (a b c) (_win_results3 "items" (merge (_win_results3 "items") (list c)))) 2)
	(assert (equal? (_win_results3 "items") '(30 40 nil nil)) true "window_flush LEAD(2) flushes 2 nils")

/* stride=2: tracking two columns, LAG(col1,1) + LAG(col2,1)
	window_size=2, stride=2 -> 4 slots, emit gets 4 flat args: old_c1 old_c2 new_c1 new_c2 */
	(define _win_results4 (newsession))
	(_win_results4 "items" '())
	(define _win_stride2 (list 0 0 2 nil nil nil nil))
	(set _win_stride2 (window_mut _win_stride2 (lambda (old_c1 old_c2 new_c1 new_c2) (_win_results4 "items" (merge (_win_results4 "items") (list old_c1 old_c2)))) (list 10 100)))
	(assert (equal? (_win_results4 "items") '(nil nil)) true "window_mut stride=2 row1 emits old=(nil nil)")
	(set _win_stride2 (window_mut _win_stride2 (lambda (old_c1 old_c2 new_c1 new_c2) (_win_results4 "items" (merge (_win_results4 "items") (list old_c1 old_c2)))) (list 20 200)))
	(assert (equal? (_win_results4 "items") '(nil nil 10 100)) true "window_mut stride=2 row2 emits old=(10 100)")
	(set _win_stride2 (window_mut _win_stride2 (lambda (old_c1 old_c2 new_c1 new_c2) (_win_results4 "items" (merge (_win_results4 "items") (list old_c1 old_c2)))) (list 30 300)))
	(assert (equal? (_win_results4 "items") '(nil nil 10 100 20 200)) true "window_mut stride=2 row3 emits old=(20 200)")

	/* promise */
	(print "testing promise ...")
	(define p1 (newpromise))
	(assert (nil? (p1 "value")) true "unresolved promise value is nil")
	(assert (nil? (p1 "state")) true "unresolved promise state is nil")
	(define p2 (newpromise))
	(p2 "value" 42)
	(assert (equal? (p2 "value") 42) true "resolved promise returns stored value")
	(assert (equal? (p2 "state") true) true "resolved promise state is true")
	(define p3 (newpromise))
	(p3 "value" 1)
	(p3 "value" 2)
	(assert (equal? (p3 "value") 2) true "second resolution overwrites first")
	(define p4 (newpromise))
	(p4 "value" 5)
	(p4 "fail")
	(assert (equal? (p4 "state") false) true "failed promise state is false")
	(assert (nil? (p4 "value")) true "failed promise without payload clears value")
	(define p5 (newpromise))
	(p5 "fail" "boom")
	(assert (equal? (p5 "state") false) true "failed promise with payload keeps failed state")
	(assert (equal? (p5 "value") "boom") true "failed promise stores payload")
	(define p6 (newpromise))
	(context (lambda () (p6 "value" 99)))
	(assert (equal? (p6 "value") 99) true "promise resolves from inside context")
	(define p7 (newpromise))
	(setTimeout (lambda () (p7 "value" "async")) 1)
	(context (lambda () (sleep 0.02)))
	(assert (equal? (p7 "value") "async") true "promise resolves from async callback")
	/* once: resolve exactly once, panic on second */
	(define p8 (newpromise))
	(p8 "once" 77)
	(assert (equal? (p8 "value") 77) true "once resolves value")
	(assert (equal? (try (lambda () (begin (p8 "once" 88) false)) (lambda (e) true)) true) true "once panics on second call")
	/* once with custom error message */
	(define p9 (newpromise))
	(p9 "once" 1)
	(assert (equal? (try (lambda () (begin (p9 "once" 2 "scalar subselect returned more than one row") false)) (lambda (e) (equal? e "scalar subselect returned more than one row"))) true) true "once custom error message")
	/* promise interface error handling */
	(define p10 (newpromise))
	(assert (equal? (try (lambda () (p10)) (lambda (e) "caught")) "caught") true "promise: 0 args panics")
	(assert (equal? (try (lambda () (p10 "nonexistent")) (lambda (e) "caught")) "caught") true "promise: unknown operation panics")
	(assert (equal? (try (lambda () (p10 "value" 1 2 3)) (lambda (e) "caught")) "caught") true "promise: too many args panics")

	/* newsession interface tests */
	(print "testing session interface ...")
	(define s1 (newsession))
	(assert (equal? (s1) (list)) true "session: empty session lists no keys")
	(s1 "x" 42)
	(assert (equal? (s1 "x") 42) true "session: get returns stored value")
	(s1 "y" "hello")
	(assert (equal? (s1 "y") "hello") true "session: get returns string value")
	(assert (nil? (s1 "missing")) true "session: nonexistent key returns nil")
	(s1 "x" 99)
	(assert (equal? (s1 "x") 99) true "session: overwrite works")
	(define s1_keys (s1))
	(assert (contains? s1_keys "x") true "session: lists key x")
	(assert (contains? s1_keys "y") true "session: lists key y")
	(define s1_compute_calls (newsession))
	(define s1_scope_a (newsession))
	(define s1_compute_values (parallelN 8 (lambda (i)
		(s1 "get_or_compute_scoped" s1_scope_a "shared" (lambda () (begin
			(s1_compute_calls (string i) true)
			123))))))
	(assert s1_compute_values '(123 123 123 123 123 123 123 123) "session: scoped compute shares producer value")
	(assert (count (s1_compute_calls)) 1 "session: scoped compute runs one producer")
	(define s1_scope_b (newsession))
	(assert (s1 "get_or_compute_scoped" s1_scope_a "shared" (lambda () 7)) 123 "session: scoped compute reuses first scope")
	(assert (s1 "get_or_compute_scoped" s1_scope_b "value" (lambda () 9)) 9 "session: scoped compute isolates scopes")
	(assert (equal? (try (lambda () (s1 "a" "b" "c")) (lambda (e) "caught")) "caught") true "session: too many args panics")

	/* deep callback signature validation */
	(print "testing callback signature validation ...")
	/* too many params rejected (lambda declares more params than caller provides) */
	(assert (equal? (try (lambda () (eval '(filter '(1 2 3) (lambda (x y) true)))) (lambda (e) "caught")) "caught") true "validate: filter callback with 2 params rejected (expects max 1)")
	(assert (equal? (try (lambda () (eval '(map '(1 2 3) (lambda (x y z) 1)))) (lambda (e) "caught")) "caught") true "validate: map callback with 3 params rejected (expects max 1)")
	(assert (equal? (try (lambda () (eval '(reduce '(1 2 3) (lambda (a b c) a) 0))) (lambda (e) "caught")) "caught") true "validate: reduce callback with 3 params rejected (expects max 2)")
	/* fewer params is valid (excess args silently ignored in this dialect) */
	(assert (equal? (filter '(1 2 3) (lambda () true)) '(1 2 3)) true "validate: filter callback with 0 params accepted")
	/* correct callbacks pass validation */
	(assert (equal? (filter '(1 2 3) (lambda (x) (> x 1))) '(2 3)) true "validate: filter with correct callback works")
	(assert (equal? (map '(1 2 3) (lambda (x) (+ x 10))) '(11 12 13)) true "validate: map with correct callback works")
	(assert (equal? (reduce '(1 2 3) (lambda (a b) (+ a b)) 0) 6) true "validate: reduce with correct callback works")

	/* return-type propagation: optimizer preserves newpromise/newsession through optimization */
	(print "testing return-type propagation ...")
	/* Verify that newpromise/newsession survive optimization and remain callable */
	(define rtp (newpromise))
	(rtp "value" 42)
	(assert (equal? (rtp "value") 42) true "return-type propagation: newpromise callable after define")
	(define rts (newsession))
	(rts "k" 99)
	(assert (equal? (rts "k") 99) true "return-type propagation: newsession callable after define")

	/* dashboard metrics (metrics.go) */
	(print "testing dashboard metrics ...")
	(define _stat (stat))
	(assert (> (_stat "mem_total") 0) true "stat mem_total returns positive")
	(assert (> (_stat "mem_available") 0) true "stat mem_available returns positive")
	(assert (<= (_stat "mem_available") (_stat "mem_total")) true "stat mem_available <= mem_total")
	(assert (> (_stat "process_memory") 0) true "stat process_memory returns positive")
	(assert (>= (cpu_usage) 0) true "cpu_usage >= 0")
	(assert (<= (cpu_usage) 100.1) true "cpu_usage <= 100")
	(assert (>= (active_connections) 0) true "active_connections >= 0")
	(assert (>= (max_connections) 1) true "max_connections >= 1")
	(assert (>= (requests_per_second) 0) true "requests_per_second >= 0")

	/* Cachemap */
	(print "testing cachemap ...")
	(define cm (newcachemap))
	/* set and get */
	(cm "k1" 42)
	(assert (cm "k1") 42 "cachemap set/get returns value")
	/* overwrite existing key */
	(cm "k1" 99)
	(assert (cm "k1") 99 "cachemap overwrite returns new value")
	/* nonexistent key returns nil */
	(assert (cm "missing") nil "cachemap get nonexistent returns nil")
	/* multiple entries */
	(cm "a" "hello")
	(cm "b" "world")
	(assert (cm "a") "hello" "cachemap get a")
	(assert (cm "b") "world" "cachemap get b")
	/* list keys returns all set keys */
	(define cm_keys (cm))
	(assert (contains? cm_keys "k1") true "cachemap lists key k1")
	(assert (contains? cm_keys "a") true "cachemap lists key a")
	(assert (contains? cm_keys "b") true "cachemap lists key b")
	/* concurrent misses run exactly one producer and share its value */
	(define sf_cm (newcachemap))
	(define sf_calls (newsession))
	(define sf_values (parallelN 8 (lambda (i)
		(sf_cm "get_or_compute" "shared" (lambda () (begin
			(sf_calls (string i) true)
			(sleep 0.02)
			123))))))
	(assert sf_values '(123 123 123 123 123 123 123 123) "cachemap singleflight shares producer value")
	(assert (count (sf_calls)) 1 "cachemap singleflight runs one producer")
	/* failed producers are not cached and a later caller can retry */
	(assert (try
		(lambda () (begin (sf_cm "get_or_compute" "retry" (lambda () (car (sf_calls "missing")))) false))
		(lambda (e) true)) true "cachemap singleflight forwards producer panic")
	(assert (sf_cm "get_or_compute" "retry" (lambda () 77)) 77 "cachemap singleflight retries after panic")

	/* query_expr_alias_set is the planner JIT coverage target. Native compilation
	is atomic, so a compiled descriptor means every expression in the function was
	lowered without a whole-procedure fallback. Exercise representative match paths,
	the nested reduce lambda, recursion, and both resolve_column_alias branches. */
	(set resolve_column_alias (jit resolve_column_alias))
	(set query_expr_alias_set (jit query_expr_alias_set))
	(assert (jit? resolve_column_alias) (jit-enabled?) "jit coverage: resolve_column_alias is 100% native")
	(assert (jit? query_expr_alias_set) (jit-enabled?) "jit coverage: query_expr_alias_set is 100% native")
	(assert (query_expr_alias_set 'default (list 'get_column 'a 'x nil nil) '()) (list 'a true)
		"jit coverage: symbol get_column match")
	(assert (query_expr_alias_set 'default
		(list '+ (list 'get_column 'a 'x nil nil) (list 'get_column nil 'y nil nil)) '())
		(list 'a true 'default true)
		"jit coverage: cons recursion and nested reduce lambda")
	(assert (query_expr_alias_set 'default 42 (list 'existing true)) (list 'existing true)
		"jit coverage: scalar fallback match")

	(print "finished unit tests")
	(print "test result: " (teststat "success") "/" (teststat "count"))
	(if (< (teststat "success") (teststat "count")) (begin
		(print "")
		(print "---- !!! some test cases have failed !!! ----")
		(print "")
		(print " it is unsafe to run memcp in this configuration")
	) (print "all tests succeeded."))
	(print "")
) /* end enclosure */
