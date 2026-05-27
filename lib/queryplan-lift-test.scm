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
== Tests for lib/queryplan-lift.scm — Phase 1 ==

Covers:
  - Marker detection: qpl-marker-kind, qpl-marker?, qpl-collect-markers,
    qpl-marker-subquery, qpl-marker-lhs
  - qpl-tuple-has-markers? on various tuples
  - lift_dep_joins_pass for the two Phase 1 paths:
      (a) no markers → qpir-leaf
      (b) single scalar inner_select in SELECT-list → qpir-dep-join wrapping
          two qpir-leaf children
  - Errors-loudly behavior: unsupported shapes panic with descriptive message
    (verified via try/catch)

Runs at server startup after queryplan-lift.scm is loaded.
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

	/* ==== qpl-marker-kind ==== */
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

	/* ==== qpl-marker-subquery + qpl-marker-lhs ==== */
	(qpl-assert (qpl-marker-subquery marker-scalar) sub-pi "subquery extract scalar")
	(qpl-assert (qpl-marker-subquery marker-in) sub-pi "subquery extract in")
	(qpl-assert (qpl-marker-subquery marker-exists) sub-pi "subquery extract exists")
	(qpl-assert (qpl-marker-lhs marker-in) (mk-col "po" "k") "lhs extract in")
	(qpl-assert (qpl-marker-lhs marker-scalar) nil "lhs scalar returns nil")
	(qpl-assert (qpl-marker-lhs marker-exists) nil "lhs exists returns nil")

	/* ==== qpl-collect-markers ==== */
	/* Atom — no markers */
	(qpl-assert (count (qpl-collect-markers 42)) 0 "collect: atom yields 0")
	(qpl-assert (count (qpl-collect-markers (mk-col "po" "k"))) 0 "collect: get_column yields 0")
	/* Bare marker — exactly one */
	(qpl-assert (count (qpl-collect-markers marker-scalar)) 1 "collect: bare scalar marker yields 1")
	/* Marker nested inside another expression */
	(define wrapped (list (quote +) (mk-col "po" "id") marker-scalar))
	(qpl-assert (count (qpl-collect-markers wrapped)) 1 "collect: wrapped marker yields 1")
	/* Two markers in same expression */
	(define two-marks (list (quote and) marker-in marker-exists))
	(qpl-assert (count (qpl-collect-markers two-marks)) 2 "collect: two siblings yields 2")

	/* ==== qpl-tuple-has-markers? ==== */
	(define t-pure (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id")))
		true))
	(qpl-assert (qpl-tuple-has-markers? t-pure) false "pure tuple has no markers")

	(define t-scalar (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))
			(list "total" marker-scalar))
		true))
	(qpl-assert (qpl-tuple-has-markers? t-scalar) true "tuple with scalar marker in field is detected")

	(define t-in-where (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id")))
		marker-in))
	(qpl-assert (qpl-tuple-has-markers? t-in-where) true "tuple with IN-marker in WHERE is detected")

	/* ==== qpl-marker-is-scalar-list-field? ==== */
	(qpl-assert (qpl-marker-is-scalar-list-field? t-pure) false "scalar-list shape: pure tuple → false")
	(qpl-assert (qpl-marker-is-scalar-list-field? t-scalar) true "scalar-list shape: t-scalar → true")
	(qpl-assert (qpl-marker-is-scalar-list-field? t-in-where) false "scalar-list shape: WHERE marker → false")

	/* ==== lift_dep_joins_pass — path (a) no markers → qpir-leaf ==== */
	(define lifted-pure (lift_dep_joins_pass t-pure))
	(qpl-assert (qpir-kind lifted-pure) (quote qpir-leaf) "lift: pure tuple → qpir-leaf")
	(qpl-assert (qpir-leaf-7tuple lifted-pure) t-pure "lift: leaf preserves tuple")

	/* ==== lift_dep_joins_pass — path (b) scalar SELECT-list lift → qpir-dep-join ==== */
	(define lifted-scalar (lift_dep_joins_pass t-scalar))
	(qpl-assert (qpir-kind lifted-scalar) (quote qpir-dep-join) "lift: scalar marker → qpir-dep-join")
	(qpl-assert (qpir-dep-join-predicate lifted-scalar) true "lift: dep-join predicate = true (correlation in inner only)")
	(qpl-assert (qpir-kind (qpir-dep-join-left lifted-scalar)) (quote qpir-leaf) "lift: left is qpir-leaf")
	(qpl-assert (qpir-kind (qpir-dep-join-right lifted-scalar)) (quote qpir-leaf) "lift: right is qpir-leaf")

	/* Outer leaf: the marker field's expression is now a get_column on a fresh sq alias */
	(define outer-tuple (qpir-leaf-7tuple (qpir-dep-join-left lifted-scalar)))
	(define outer-total-pair (nth (qpp-tuple-fields outer-tuple) 1))
	(define outer-total-expr (nth outer-total-pair 1))
	(qpl-assert (qpir-kind outer-tuple) nil "outer 7-tuple is not an IR node")
	(qpl-assert (nth outer-total-pair 0) "total" "outer field name preserved")
	/* shape: (get_column sq_? false "value" false) */
	(qpl-assert (nth outer-total-expr 0) (quote get_column) "outer marker replaced by get_column")
	(qpl-assert (nth outer-total-expr 3) "value" "outer get_column refers to value column")

	/* Inner leaf: subquery's single field is renamed to "value" */
	(define inner-tuple (qpir-leaf-7tuple (qpir-dep-join-right lifted-scalar)))
	(qpl-assert (count (qpp-tuple-fields inner-tuple)) 1 "inner subquery has 1 field")
	(qpl-assert (nth (nth (qpp-tuple-fields inner-tuple) 0) 0) "value" "inner subquery field renamed to value")

	/* ==== Errors-loudly: unsupported WHERE-level marker triggers error ==== */
	(define caught-where (try
		(lambda () (begin (lift_dep_joins_pass t-in-where) "no-error"))
		(lambda (e) "errored")))
	(qpl-assert caught-where "errored" "lift: WHERE-level marker triggers error (Phase 1 unsupported)")

	/* ==== Errors-loudly: non-tuple input ==== */
	(define caught-nontuple (try
		(lambda () (begin (lift_dep_joins_pass (list 1 2 3)) "no-error"))
		(lambda (e) "errored")))
	(qpl-assert caught-nontuple "errored" "lift: non-tuple input triggers error")

	(print "  qpl tests: " (- (qpl-tests "count") (qpl-tests "fail")) "/" (qpl-tests "count") " passed")
	(if (> (qpl-tests "fail") 0) (begin
		(print "")
		(print "  !!! queryplan-lift self-tests failed !!!")
		(print "  it is unsafe to run memcp in this configuration")
	) nil))
