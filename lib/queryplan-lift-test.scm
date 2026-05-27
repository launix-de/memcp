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
  - Errors-loudly for: HAVING markers, GROUP-BY markers, ORDER-BY markers,
    inner_select_in, inner_select_exists, non-tuple input

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
	/* The (coalesce sq.value 0) shape */
	(qpl-assert (nth (nth in-pred 1) 0) (quote coalesce) "lift: IN rewrite wraps with coalesce")
	(qpl-assert (nth (nth in-pred 1) 2) 0 "lift: IN rewrite coalesce default = 0")
	(define in-sq-ref (nth (nth in-pred 1) 1))
	(qpl-assert (nth in-sq-ref 0) (quote get_column) "lift: IN rewrite coalesce arg = get_column on sq alias")
	(qpl-assert (nth in-sq-ref 3) "value" "lift: IN rewrite coalesce arg points to value column")
	(qpl-assert (qpir-kind (qpir-select-child lifted-in)) (quote qpir-dep-join)
		"lift: IN-rewrite chain has qpir-dep-join under select")

	/* ==== Path (f): EXISTS-marker in WHERE rewrites the same way ==== */
	(define t-exists (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id")))
		marker-exists))
	(define lifted-exists (lift_dep_joins_pass t-exists))
	(qpl-assert (qpir-kind lifted-exists) (quote qpir-select)
		"lift: EXISTS-in-WHERE → qpir-select at root")
	(define ex-pred (qpir-select-predicate lifted-exists))
	(qpl-assert (nth ex-pred 0) (quote >) "lift: EXISTS rewrite uses > comparison")
	(qpl-assert (nth (nth ex-pred 1) 0) (quote coalesce) "lift: EXISTS rewrite uses coalesce")
	(qpl-assert (qpir-kind (qpir-select-child lifted-exists)) (quote qpir-dep-join)
		"lift: EXISTS-rewrite chain has qpir-dep-join under select")

	/* ==== Decomposition verification for FAQ §11 COUNT subqueries ==== */
	/* IN rewrite produces an inner COUNT(*) subquery — phase 4 decomposes it
	   into qpir-groupby (empty keys, one COUNT agg). Since sub-pi has a
	   correlated WHERE, phase 5 also wraps the inner leaf with qpir-select. */
	(define ex-dj (qpir-select-child lifted-exists))
	(define ex-right (qpir-dep-join-right ex-dj))
	(qpl-assert (qpir-kind ex-right) (quote qpir-groupby)
		"EXISTS rewrite: right of dep-join is qpir-groupby (COUNT decomposed)")
	(qpl-assert (count (qpir-groupby-keys ex-right)) 0
		"EXISTS rewrite groupby: empty keys (static group)")
	(qpl-assert (count (qpir-groupby-aggs ex-right)) 1
		"EXISTS rewrite groupby: exactly one aggregate")
	(qpl-assert (qpir-kind (qpir-groupby-child ex-right)) (quote qpir-select)
		"EXISTS rewrite groupby child is qpir-select (correlated WHERE hoisted)")
	(qpl-assert (qpir-kind (qpir-select-child (qpir-groupby-child ex-right))) (quote qpir-leaf)
		"EXISTS rewrite bottom is qpir-leaf")

	/* ==== Errors-loudly ==== */

	(define t-having-marker (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))) true '() marker-scalar '() nil nil))
	(qpl-assert (try
		(lambda () (begin (lift_dep_joins_pass t-having-marker) "no-error"))
		(lambda (e) "errored")) "errored"
		"lift: HAVING marker triggers error in Phase 2")

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
