/*
Copyright (C) 2026  Carl-Philip Hänsch

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
== Tests for lib/queryplan-lift.scm — Phase 2 ==

Covers:
- Marker detection helpers (kind, ?, collect, subquery, lhs)
- qpl-tuple-has-markers? on various tuples
- lift_dep_joins_pass paths:
(a) no markers → qpir-leaf
(b) single scalar marker in SELECT-list → qpir-dep-join
(c) single scalar marker in WHERE → qpir-select(qpir-dep-join …)
(d) two scalar markers (one in field, one in WHERE) → chained dep-joins
wrapped by qpir-select
- HAVING markers are lifted like field markers
- Errors-loudly for: GROUP-BY markers, ORDER-BY markers, non-tuple input

Runs at server startup after queryplan-lift.scm loads.
*/

(begin
	(print "testing queryplan-lift ...")
	(define qpl-tests (newsession))
	(qpl-tests "count" 0)
	(qpl-tests "fail" 0)
	(define qpl-assert (lambda (val expected errormsg) (begin
		(qpl-tests "count" (+ (qpl-tests "count") 1))
		(if (equal? val expected)
			nil
			(begin
				(qpl-tests "fail" (+ (qpl-tests "fail") 1))
				(print "  qpl-test FAIL: " errormsg " (got: " val ", expected: " expected ")"))))))

	(define mk-col (lambda (tv col) (list (quote get_column) tv false col false)))
	(define mk-tuple (lambda (schema tables fields cond) (list schema tables fields cond '() nil '() nil nil)))
	(define qpl-count-dep-joins (lambda (node) (begin
		(define self-count (if (equal? (qpir-kind node) (quote qpir-dep-join)) 1 0))
		(define child-count (reduce (qpir-children node)
			(lambda (acc c) (+ acc (qpl-count-dep-joins c))) 0))
		(+ self-count child-count))))
	(define qpl-tree-has-inner-select-marker? (lambda (expr)
		(match expr
			'((symbol inner_select) _) true
			'((quote inner_select) _) true
			'((symbol inner_select_exists) _) true
			'((quote inner_select_exists) _) true
			'((symbol inner_select_in) _ _) true
			'((quote inner_select_in) _ _) true
			(cons head args) (reduce (coalesceNil args '()) (lambda (acc a)
				(or acc (qpl-tree-has-inner-select-marker? a))) false)
			false)))

	/* ==== Marker detection ==== */
	(define sub-pi (mk-tuple "memcp-tests"
		(list (list "pi" "memcp-tests" "pi" false nil))
		(list (list "total" (list (quote aggregate) (mk-col "pi" "amount") (quote +) 0)))
		(list (quote equal??) (mk-col "pi" "k") (mk-col "po" "k"))))
	(define marker-scalar (list (quote inner_select) sub-pi))
	(define marker-in (list (quote inner_select_in) (mk-col "po" "k") sub-pi))
	(define marker-exists (list (quote inner_select_exists) sub-pi))
	(qpl-assert (qpl-marker-kind marker-scalar) (quote inner_select) "marker-kind scalar")
	(qpl-assert (qpl-marker-kind marker-in) (quote inner_select_in) "marker-kind in")
	(qpl-assert (qpl-marker-kind marker-exists) (quote inner_select_exists) "marker-kind exists")
	(qpl-assert (qpl-marker-kind (mk-col "po" "k")) nil "marker-kind non-marker returns nil")
	(qpl-assert (qpl-marker-kind 42) nil "marker-kind atom returns nil")
	(qpl-assert (qpl-marker? marker-scalar) true "qpl-marker? true on scalar")
	(qpl-assert (qpl-marker? (mk-col "po" "k")) false "qpl-marker? false on get_column")
	(qpl-assert (qpl-marker-subquery marker-scalar) sub-pi "subquery extract scalar")
	(qpl-assert (qpl-marker-subquery marker-in) sub-pi "subquery extract in")
	(qpl-assert (qpl-marker-subquery marker-exists) sub-pi "subquery extract exists")
	(qpl-assert (qpl-marker-lhs marker-in) (mk-col "po" "k") "lhs extract in")
	(qpl-assert (qpl-marker-lhs marker-scalar) nil "lhs scalar returns nil")
	(qpl-assert (qpl-marker-lhs marker-exists) nil "lhs exists returns nil")
	(define scalar-canonical-map (build_occurrence_alias_map
		(list (list "cfg" "memcp-tests" "cd_cfg" false nil))))
	(qpl-assert (qpl-canonical-scalar-expr (mk-col "cfg" "default_kind") scalar-canonical-map)
		"memcp-tests.cd_cfg.default_kind"
		"canonical scalar names use physical source and column names")

	(qpl-assert (count (qpl-collect-markers 42)) 0 "collect: atom yields 0")
	(qpl-assert (count (qpl-collect-markers (mk-col "po" "k"))) 0 "collect: get_column yields 0")
	(qpl-assert (count (qpl-collect-markers marker-scalar)) 1 "collect: bare scalar yields 1")
	(qpl-assert (count (qpl-collect-markers (list (quote +) (mk-col "po" "id") marker-scalar))) 1
		"collect: wrapped marker yields 1")
	(qpl-assert (count (qpl-collect-markers (list (quote and) marker-in marker-exists))) 2
		"collect: two siblings yields 2")

	/* ==== qpl-tuple-has-markers? ==== */
	(define t-pure (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id")))
		true))
	(qpl-assert (qpl-tuple-has-markers? t-pure) false "pure tuple → no markers")

	(define t-scalar (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))
			(list "total" marker-scalar))
		true))
	(qpl-assert (qpl-tuple-has-markers? t-scalar) true "scalar marker in field detected")

	(define t-where-scalar (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id")))
		(list (quote >) marker-scalar 100)))
	(qpl-assert (qpl-tuple-has-markers? t-where-scalar) true "scalar marker in WHERE detected")

	(define t-in-where (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id")))
		marker-in))
	(qpl-assert (qpl-tuple-has-markers? t-in-where) true "IN-marker in WHERE detected")

	/* ==== Path (a): no markers → qpir-leaf ==== */
	(define lifted-pure (lift_dep_joins_pass t-pure))
	(qpl-assert (qpir-kind lifted-pure) (quote qpir-leaf) "lift: pure tuple → qpir-leaf")
	(qpl-assert (qpir-leaf-7tuple lifted-pure) t-pure "lift: leaf preserves tuple")

	/* ==== Path (b): scalar SELECT-list marker → qpir-dep-join with qpir-groupby-decomposed inner ==== */
	(define lifted-scalar (lift_dep_joins_pass t-scalar))
	(qpl-assert (qpir-kind lifted-scalar) (quote qpir-dep-join)
		"lift: scalar field marker (WHERE=true) → qpir-dep-join at root")
	(qpl-assert (qpir-dep-join-predicate lifted-scalar) true "lift: dep-join predicate = true")
	(qpl-assert (qpir-kind (qpir-dep-join-left lifted-scalar)) (quote qpir-leaf) "lift: left is qpir-leaf")
	(qpl-assert (qpir-kind (qpir-dep-join-right lifted-scalar)) (quote qpir-groupby)
		"lift: right of dep-join is qpir-groupby (inner SUM decomposed per FAQ §33)")

	(define outer-tuple (qpir-leaf-7tuple (qpir-dep-join-left lifted-scalar)))
	(define outer-total-pair (nth (qpp-tuple-fields outer-tuple) 1))
	(qpl-assert (nth outer-total-pair 0) "total" "outer: field name preserved")
	(qpl-assert (nth (nth outer-total-pair 1) 0) (quote get_column) "outer: marker replaced by get_column")
	(qpl-assert (nth (nth outer-total-pair 1) 3) "value" "outer: get_column refers to value column")
	(qpl-assert (qpp-tuple-condition outer-tuple) true "outer: WHERE is true")

	/* The qpir-groupby on the right has empty keys (static-group), one agg named "value",
	and a qpir-leaf below that projects the agg's inner expression as "value". */
	(define inner-gb (qpir-dep-join-right lifted-scalar))
	(qpl-assert (count (qpir-groupby-keys inner-gb)) 0 "inner groupby: empty keys (static group)")
	(qpl-assert (count (qpir-groupby-aggs inner-gb)) 1 "inner groupby: one aggregate")
	(qpl-assert (nth (nth (qpir-groupby-aggs inner-gb) 0) 0) "value" "inner groupby agg name = value")
	(qpl-assert (qpir-groupby-having inner-gb) nil "inner groupby: no HAVING")
	/* Phase 5 hoists the inner WHERE to a qpir-select between groupby and leaf
	(the WHERE in sub-pi references po.k — must be operator-level so the
	§3.3 select rule can fire during unnest). */
	(qpl-assert (qpir-kind (qpir-groupby-child inner-gb)) (quote qpir-select)
		"inner groupby child is qpir-select (WHERE hoisted)")
	(define inner-select-pred (qpir-select-predicate (qpir-groupby-child inner-gb)))
	(qpl-assert (nth inner-select-pred 0) (quote equal??)
		"inner select predicate is the original equal?? from sub-pi's WHERE")
	(define inner-bottom-leaf (qpir-select-child (qpir-groupby-child inner-gb)))
	(qpl-assert (qpir-kind inner-bottom-leaf) (quote qpir-leaf) "inner bottom is qpir-leaf")
	(define inner-leaf-tuple (qpir-leaf-7tuple inner-bottom-leaf))
	/* Leaf projects every physical pi column referenced by the aggregate's inner
	expression OR by the hoisted WHERE — phase 5/6 keeps canonical source-column
	names per FAQ. For SUM(pi.amount) WHERE pi.k=po.k that's (amount) + (k). */
	(qpl-assert (count (qpp-tuple-fields inner-leaf-tuple)) 2
		"inner leaf projects 2 columns (amount from agg, k from WHERE)")
	(qpl-assert (qpp-tuple-condition inner-leaf-tuple) true
		"inner leaf's WHERE is true (real WHERE was hoisted to qpir-select)")
	(qpl-assert (count (qpp-tuple-group inner-leaf-tuple)) 0 "inner leaf has no GROUP BY (moved up)")
	/* Aggregate inside qpir-groupby keeps its ORIGINAL physical-column reference
	(no synthesized "" placeholder). */
	(define inner-agg-pair (nth (qpir-groupby-aggs inner-gb) 0))
	(define inner-agg-expr (nth inner-agg-pair 1))
	(qpl-assert (nth inner-agg-expr 0) (quote aggregate) "inner agg is bare (aggregate …) form")
	(define inner-agg-inner-arg (nth inner-agg-expr 1))
	(qpl-assert (nth inner-agg-inner-arg 0) (quote get_column)
		"inner agg's inner expr is a (get_column …) ref")
	(qpl-assert (nth inner-agg-inner-arg 1) "pi"
		"inner agg references the physical source alias (pi), not a synthesized placeholder")

	/* ==== End-to-end F(N) check: lifted correlated SUM has F(root) = ∅ ==== */
	/* The whole architectural point of phases 4 + 5: after lift, the dep-join
	binds the outer column references inside the inner subtree. F(root) MUST
	be empty — every column ref is bound by some provider in the tree. */
	(define fv-after-lift (qpir-free-vars lifted-scalar))
	(qpl-assert (count fv-after-lift) 0
		"F(lifted dep-join) = ∅ — outer refs in inner are bound by dep-join's left provider")

	/* ==== Duplicate scalar markers share one canonical helper ==== */
	(define t-duplicate-scalar (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "total_a" marker-scalar)
			(list "total_b" marker-scalar))
		true))
	(define lifted-duplicate-scalar (lift_dep_joins_pass t-duplicate-scalar))
	(qpl-assert (qpl-count-dep-joins lifted-duplicate-scalar) 1
		"lift: duplicate canonical scalar markers share one dep-join helper")
	(define duplicate-outer (qpir-leaf-7tuple (qpir-dep-join-left lifted-duplicate-scalar)))
	(define duplicate-a-ref (nth (nth (qpp-tuple-fields duplicate-outer) 0) 1))
	(define duplicate-b-ref (nth (nth (qpp-tuple-fields duplicate-outer) 1) 1))
	(qpl-assert duplicate-a-ref duplicate-b-ref
		"lift: duplicate scalar fields reference the same canonical helper alias")
	(qpl-assert (nth duplicate-a-ref 1) (qpir-dep-join-rhs-alias lifted-duplicate-scalar)
		"lift: duplicate scalar helper alias is the dep-join rhs alias")

	/* ==== Path (c): scalar WHERE marker → qpir-select(qpir-dep-join …) ==== */
	(define lifted-where (lift_dep_joins_pass t-where-scalar))
	(qpl-assert (qpir-kind lifted-where) (quote qpir-select)
		"lift: scalar WHERE marker → qpir-select at root")
	(define inner-of-select (qpir-select-child lifted-where))
	(qpl-assert (qpir-kind inner-of-select) (quote qpir-dep-join)
		"lift: child of qpir-select is qpir-dep-join")
	/* qpir-select's predicate must reference the sq alias's value column */
	(define wpred (qpir-select-predicate lifted-where))
	(qpl-assert (nth wpred 0) (quote >) "lift: select predicate keeps original operator")
	(qpl-assert (nth (nth wpred 1) 0) (quote get_column)
		"lift: select predicate's marker arg replaced by get_column")
	(qpl-assert (nth (nth wpred 1) 3) "value" "lift: select predicate's arg points to value column")

	/* The qpir-dep-join under the select: left's WHERE must be true (we moved
	the WHERE up into the qpir-select wrapper). */
	(define outer-where-tuple (qpir-leaf-7tuple (qpir-dep-join-left inner-of-select)))
	(qpl-assert (qpp-tuple-condition outer-where-tuple) true
		"lift: outer leaf's WHERE was replaced by true (real WHERE is above the dep-join)")

	/* ==== Path (d): TWO scalar markers (one in field, one in WHERE) → chain ==== */
	/* Build a second sub-7tuple to give the markers distinct content */
	(define sub-other (mk-tuple "memcp-tests"
		(list (list "qx" "memcp-tests" "qx" false nil))
		(list (list "n" (list (quote aggregate) (mk-col "qx" "x") (quote +) 0)))
		true))
	(define marker-other (list (quote inner_select) sub-other))
	(define t-both (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))
			(list "total" marker-scalar))
		(list (quote >) marker-other 5)))
	(define lifted-both (lift_dep_joins_pass t-both))
	(qpl-assert (qpir-kind lifted-both) (quote qpir-select) "lift: two markers → qpir-select at root")
	(define dj1 (qpir-select-child lifted-both))
	(qpl-assert (qpir-kind dj1) (quote qpir-dep-join) "lift: select-child is qpir-dep-join (outermost)")
	(define dj1-left (qpir-dep-join-left dj1))
	(qpl-assert (qpir-kind dj1-left) (quote qpir-dep-join) "lift: chained: left is another qpir-dep-join")
	(qpl-assert (qpir-kind (qpir-dep-join-left dj1-left)) (quote qpir-leaf)
		"lift: chain's bottom-left is the outer qpir-leaf")

	/* ==== Path (e): IN-marker in WHERE rewrites via FAQ §11 COUNT, then lifts ==== */
/* t-in-where = SELECT po.id FROM po WHERE po.k IN (SELECT pi.amount sum FROM pi WHERE pi.k=po.k)
	After phase 3 rewrite: WHERE becomes (> (coalesce <count-subquery> 0) 0).
	The count-subquery is a scalar inner_select; phase 2 then lifts it as a dep-join. */
	(define lifted-in (lift_dep_joins_pass t-in-where))
	(qpl-assert (qpir-kind lifted-in) (quote qpir-select)
		"lift: IN-in-WHERE → qpir-select at root (COUNT scalar pulled up)")
	(define in-pred (qpir-select-predicate lifted-in))
	(qpl-assert (nth in-pred 0) (quote >) "lift: IN rewrite uses > comparison")
	(qpl-assert (nth in-pred 2) 0 "lift: IN rewrite compares to 0")
	/* The (coalesce helper.value 0) shape */
	(qpl-assert (nth (nth in-pred 1) 0) (quote coalesce) "lift: IN rewrite wraps with coalesce")
	(qpl-assert (nth (nth in-pred 1) 2) 0 "lift: IN rewrite coalesce default = 0")
	(define in-sq-ref (nth (nth in-pred 1) 1))
	(qpl-assert (nth in-sq-ref 0) (quote get_column) "lift: IN rewrite coalesce arg = get_column on sq alias")
	(qpl-assert (nth in-sq-ref 3) "value" "lift: IN rewrite coalesce arg points to value column")
	(qpl-assert (qpir-kind (qpir-select-child lifted-in)) (quote qpir-dep-join)
		"lift: IN-rewrite chain has qpir-dep-join under select")

		/* ==== Path (f): simple EXISTS-marker in WHERE rewrites to boolean LIMIT 1 ==== */
		(define t-exists (mk-tuple "memcp-tests"
			(list (list "po" "memcp-tests" "po" false nil))
			(list (list "id" (mk-col "po" "id")))
			marker-exists))
		(define lifted-exists (lift_dep_joins_pass t-exists))
		(qpl-assert (qpir-kind lifted-exists) (quote qpir-select)
			"lift: EXISTS-in-WHERE → qpir-select at root")
		(define ex-pred (qpir-select-predicate lifted-exists))
		(qpl-assert (nth ex-pred 0) (quote >) "lift: EXISTS rewrite uses positive numeric guard")
		(qpl-assert (nth (nth ex-pred 1) 0) (quote coalesce) "lift: EXISTS rewrite uses numeric coalesce")
		(qpl-assert (nth (nth (nth ex-pred 1) 1) 0) (quote get_column) "lift: EXISTS rewrite reads value column directly")
		(qpl-assert (nth (nth (nth ex-pred 1) 1) 3) "value" "lift: EXISTS rewrite reads value column")
		(qpl-assert (nth (nth ex-pred 1) 2) 0 "lift: EXISTS rewrite defaults to zero")
		(qpl-assert (nth ex-pred 2) 0 "lift: EXISTS rewrite compares to zero")
		(qpl-assert (qpir-kind (qpir-select-child lifted-exists)) (quote qpir-dep-join)
			"lift: EXISTS-rewrite chain has qpir-dep-join under select")

		/* ==== Decomposition verification for simple EXISTS subqueries ==== */
		/* A simple EXISTS does not need COUNT(*). It is represented as a
		LIMIT-1 value domain; lowering recognizes the payload and collapses
		the domain by outer keys to preserve semi-join cardinality. */
		(define ex-dj (qpir-select-child lifted-exists))
		(define ex-right (qpir-dep-join-right ex-dj))
		(qpl-assert (qpir-kind ex-right) (quote qpir-select)
			"EXISTS rewrite: right of dep-join hoists WHERE into qpir-select")
		(qpl-assert (qpir-kind (qpir-select-child ex-right)) (quote qpir-leaf)
			"EXISTS rewrite bottom is qpir-leaf")
		(define ex-leaf-tuple (qpir-leaf-7tuple (qpir-select-child ex-right)))
		(qpl-assert (qpp-tuple-limit ex-leaf-tuple) 1
			"EXISTS rewrite keeps LIMIT 1 for domain collapse")
		(qpl-assert (count (qpp-tuple-fields ex-leaf-tuple)) 1
			"EXISTS rewrite projects one payload field")
		(define ex-value-expr (nth (nth (qpp-tuple-fields ex-leaf-tuple) 0) 1))
		(qpl-assert (nth ex-value-expr 0) (quote if)
			"EXISTS rewrite payload is recognizable if form")
		(qpl-assert (nth ex-value-expr 2) 1
			"EXISTS rewrite payload true value is one")

	/* ==== Nested correlated subquery — recursive lift ==== */
	/* SQL: SELECT outer.id,
	(SELECT (SELECT MAX(t1.x) FROM t1 WHERE t1.a=t0.a) AS inner
	FROM t0 WHERE t0.b=outer.b) AS total
	FROM outer
	The OUTER's "total" field's marker contains an outer-inner 7tuple
	whose ONE field ("inner") is itself a marker over t1. */
	(define t1-sub (mk-tuple "memcp-tests"
		(list (list "t1" "memcp-tests" "t1" false nil))
		(list (list "inner" (list (quote aggregate) (mk-col "t1" "x") (quote +) 0)))
		(list (quote equal??) (mk-col "t1" "a") (mk-col "t0" "a"))))
	(define t0-sub (mk-tuple "memcp-tests"
		(list (list "t0" "memcp-tests" "t0" false nil))
		(list (list "inner" (list (quote inner_select) t1-sub)))
		(list (quote equal??) (mk-col "t0" "b") (mk-col "outer" "b"))))
	(define t-nested (mk-tuple "memcp-tests"
		(list (list "outer" "memcp-tests" "outer" false nil))
		(list (list "id" (mk-col "outer" "id"))
			(list "total" (list (quote inner_select) t0-sub)))
		true))
	(define lifted-nested (lift_dep_joins_pass t-nested))
	(qpl-assert (qpir-kind lifted-nested) (quote qpir-dep-join)
		"nested lift: root is qpir-dep-join (outer's marker)")
	/* Walk the tree counting qpir-dep-joins (qpu-count-dep-joins isn't
	loaded yet at this point). */
	(define dj-count (qpl-count-dep-joins lifted-nested))
	/* Outer dep-join + inner dep-join from recursive lift = 2 */
	(qpl-assert dj-count 2
		"nested lift: exactly 2 qpir-dep-joins total (outer + inner from recursive lift)")

	/* ==== Nested scalar inside aggregate input remains marker-free ==== */
	(define item-total-sub (mk-tuple "memcp-tests"
		(list (list "item" "memcp-tests" "item" false nil))
		(list (list "total" (list (quote aggregate)
			(list (quote *) (mk-col "item" "price") (mk-col "item" "qty"))
			(quote +) 0)))
		(list (quote equal??) (mk-col "item" "invoice_id") (mk-col "paid" "id"))))
	(define paid-total-sub (mk-tuple "memcp-tests"
		(list (list "paid" "memcp-tests" "invoice" false nil))
		(list (list "total" (list (quote aggregate)
			(list (quote coalesce) (list (quote inner_select) item-total-sub) 0)
			(quote +) 0)))
		(list (quote and)
			(list (quote equal??) (mk-col "paid" "customer_id") (mk-col "outer" "customer_id"))
			(list (quote not) (list (quote nil?) (mk-col "paid" "paid_at"))))))
	(define t-nested-aggregate-input (mk-tuple "memcp-tests"
		(list (list "outer" "memcp-tests" "invoice" false nil))
		(list (list "id" (mk-col "outer" "id"))
			(list "paid_total" (list (quote inner_select) paid-total-sub)))
		true))
	(define lifted-nested-aggregate-input (lift_dep_joins_pass t-nested-aggregate-input))
	(define nested-aggregate-groupby (qpir-dep-join-right lifted-nested-aggregate-input))
	(qpl-assert (qpir-kind nested-aggregate-groupby) (quote qpir-groupby)
		"nested aggregate input lift: outer scalar aggregate becomes qpir-groupby")
	(qpl-assert (qpl-tree-has-inner-select-marker? (qpir-groupby-aggs nested-aggregate-groupby)) false
		"nested aggregate input lift: groupby aggregate uses substituted scalar helper")

	/* ==== HAVING markers lift through the same dep-join chain ==== */

	(define t-having-marker (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))) true '() marker-scalar '() nil nil))
	(define lifted-having (lift_dep_joins_pass t-having-marker))
	(qpl-assert (qpir-kind lifted-having) (quote qpir-dep-join)
		"lift: HAVING marker is lifted as a dep-join")
	(qpl-assert (qpl-tuple-has-markers? (qpir-leaf-7tuple (qpir-dep-join-left lifted-having))) false
		"lift: outer leaf has no raw HAVING marker after substitution")

	/* ==== Errors-loudly ==== */

	(define t-group-marker (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))) true (list marker-scalar) nil '() nil nil))
	(qpl-assert (try
		(lambda () (begin (lift_dep_joins_pass t-group-marker) "no-error"))
		(lambda (e) "errored")) "errored"
		"lift: GROUP-BY marker triggers error in Phase 2")

	(define t-order-marker (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))) true '() nil
		(list (list marker-scalar (quote asc))) nil nil))
	(qpl-assert (try
		(lambda () (begin (lift_dep_joins_pass t-order-marker) "no-error"))
		(lambda (e) "errored")) "errored"
		"lift: ORDER-BY marker triggers error in Phase 2")

	(qpl-assert (try
		(lambda () (begin (lift_dep_joins_pass (list 1 2 3)) "no-error"))
		(lambda (e) "errored")) "errored"
		"lift: non-tuple input triggers error")

	(print "  qpl tests: " (- (qpl-tests "count") (qpl-tests "fail")) "/" (qpl-tests "count") " passed")
	(if (> (qpl-tests "fail") 0) (begin
		(print "")
		(print "  !!! queryplan-lift self-tests failed !!!")
		(print "  it is unsafe to run memcp in this configuration")
	) nil))
