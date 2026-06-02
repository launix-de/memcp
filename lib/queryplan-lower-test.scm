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
== Tests for lib/queryplan-lower.scm — Phase 1 ==

Covers:
- qpir-leaf → its 7-tuple (identity)
- qpir-select → AND predicate into condition
- qpir-map → replace fields
- qpir-groupby → wrap into a 7-tuple with group/aggs in fields
- qpir-join (no rhs-alias) → merge tables + AND
- qpir-join (with rhs-alias) → right wrapped as derived table + rewrite refs
- End-to-end pipeline + lowering on the canonical correlated SUM
- Errors-loudly for unsupported operators (qpir-dep-join, qpir-window, etc.)
*/

(begin
	(print "testing queryplan-lower ...")
	(define qpw-tests (newsession))
	(qpw-tests "count" 0)
	(qpw-tests "fail" 0)
	(define qpw-assert (lambda (val expected errormsg) (begin
		(qpw-tests "count" (+ (qpw-tests "count") 1))
		(if (equal? val expected)
			nil
			(begin
				(qpw-tests "fail" (+ (qpw-tests "fail") 1))
				(print "  qpw-test FAIL: " errormsg " (got: " val ", expected: " expected ")"))))))

	(define mk-col (lambda (tv col) (list (quote get_column) tv false col false)))
	(define mk-tuple (lambda (schema tables fields cond)
		(list schema tables fields cond (list) nil (list) nil nil)))

	/* ==== qpir-leaf → identity ==== */
	(define t-leaf-tuple (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))) true))
	(define t-leaf (qpir-leaf t-leaf-tuple))
	(define lo-leaf (lower_to_scans_pass t-leaf))
	(qpw-assert (qpp-tuple? lo-leaf) true "lower(leaf) returns a 7-tuple")
	(qpw-assert lo-leaf t-leaf-tuple "lower(leaf) is identity on the 7-tuple")

	/* ==== qpir-select → AND predicate ==== */
	(define select-pred (list (quote >) (mk-col "po" "id") 10))
	(define t-sel (qpir-select select-pred t-leaf))
	(define lo-sel (lower_to_scans_pass t-sel))
	(qpw-assert (qpp-tuple? lo-sel) true "lower(select) returns 7-tuple")
	(qpw-assert (qpp-tuple-condition lo-sel) select-pred
		"lower(select(leaf-with-true-cond)) → tuple with predicate as condition")
	(qpw-assert (qpp-tuple-tables lo-sel) (qpp-tuple-tables t-leaf-tuple)
		"lower(select) preserves child's tables")
	(qpw-assert (qpp-tuple-fields lo-sel) (qpp-tuple-fields t-leaf-tuple)
		"lower(select) preserves child's fields")

	/* ==== qpir-select over an existing non-trivial WHERE → AND ==== */
	(define t-leaf-with-where (qpir-leaf (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id")))
		(list (quote equal??) (mk-col "po" "k") 5))))
	(define t-sel2 (qpir-select select-pred t-leaf-with-where))
	(define lo-sel2 (lower_to_scans_pass t-sel2))
	(define cond2 (qpp-tuple-condition lo-sel2))
	(qpw-assert (nth cond2 0) (quote and)
		"lower(select) over existing WHERE → AND combination")

	/* ==== qpir-map → replace fields ==== */
	(define new-projs (list (list "renamed" (mk-col "po" "id"))))
	(define t-map (qpir-map new-projs t-leaf))
	(define lo-map (lower_to_scans_pass t-map))
	(qpw-assert (qpp-tuple-fields lo-map) new-projs
		"lower(map) replaces fields")

	/* ==== qpir-groupby → wraps with group + aggs as fields ==== */
	(define pi-leaf (qpir-leaf (mk-tuple "memcp-tests"
		(list (list "pi" "memcp-tests" "pi" false nil))
		(list (list "amount" (mk-col "pi" "amount")) (list "k" (mk-col "pi" "k")))
		true)))
	(define gb (qpir-groupby
		(list (mk-col "pi" "k"))
		(list (list "value"
			(list (quote aggregate) (mk-col "pi" "amount") (quote +) 0)))
		nil
		pi-leaf))
	(define lo-gb (lower_to_scans_pass gb))
	(qpw-assert (qpp-tuple? lo-gb) true "lower(groupby) returns 7-tuple")
	(qpw-assert (count (qpp-tuple-group lo-gb)) 1
		"lower(groupby): one group-by key in the tuple's group slot")
	/* Fields now include both the key-projection (k from pi.k) and the agg (value) */
	(qpw-assert (count (qpp-tuple-fields lo-gb)) 2
		"lower(groupby): fields = key projections + agg projections (2 total)")

	/* ==== qpir-join no rhs-alias → table-list merge ==== */
	(define qx-leaf (qpir-leaf (mk-tuple "memcp-tests"
		(list (list "qx" "memcp-tests" "qx" false nil))
		(list (list "x" (mk-col "qx" "x"))) true)))
	(define j-pred (list (quote equal??) (mk-col "po" "k") (mk-col "qx" "k")))
	(define j-simple (qpir-join (quote inner) j-pred t-leaf qx-leaf nil))
	(define lo-j (lower_to_scans_pass j-simple))
	(qpw-assert (count (qpp-tuple-tables lo-j)) 2
		"lower(simple join) merges left and right tables")
	(qpw-assert (qpp-tuple-condition lo-j) j-pred
		"lower(simple join) uses the join predicate as condition")

	/* ==== qpir-join WITH rhs-alias → derived-table wrap + ref retarget ==== */
	/* This is the canonical post-unnest correlated SUM shape:
	qpir-join (po.k=pi.k) rhs-alias=sq_1
	qpir-leaf po with fields [(id po.id) (total sq_1.value)]
	qpir-groupby [pi.k] [(value SUM(pi.amount))] nil
	qpir-leaf pi */
	(define outer-with-sq (qpir-leaf (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))
			(list "total" (mk-col "sq_1" "value")))
		true)))
	(define inner-gb (qpir-groupby
		(list (mk-col "pi" "k"))
		(list (list "value"
			(list (quote aggregate) (mk-col "pi" "amount") (quote +) 0)))
		nil
		pi-leaf))
	(define join-corr (qpir-join (quote inner)
		(list (quote equal??) (mk-col "po" "k") (mk-col "pi" "k"))
		outer-with-sq inner-gb "sq_1"))
	(define lo-corr (lower_to_scans_pass join-corr))
	(qpw-assert (qpp-tuple? lo-corr) true "lower(join with rhs-alias) returns 7-tuple")
	(qpw-assert (count (qpp-tuple-tables lo-corr)) 2
		"lower(join with rhs-alias): 2 tables (outer + derived)")

	/* The second table should be the derived table: (sq_1 schema sub-7-tuple false nil) */
	(define derived-entry (nth (qpp-tuple-tables lo-corr) 1))
	(qpw-assert (nth derived-entry 0) "sq_1" "derived entry alias = sq_1")
	(qpw-assert (qpp-tuple? (nth derived-entry 2)) true
		"derived entry's 3rd slot is a sub-7-tuple")

	/* The join predicate was (po.k = pi.k); after lowering, pi.k retargets to sq_1.k */
	(define cond-corr (qpp-tuple-condition lo-corr))
	(qpw-assert (nth cond-corr 0) (quote equal??) "lowered join condition starts with equal??")
	(qpw-assert (nth (nth cond-corr 2) 1) "sq_1"
		"join condition's right side retargeted from pi to sq_1")
	(qpw-assert (nth (nth cond-corr 2) 3) "k"
		"join condition's right side column is k")

	/* ==== Errors-loudly ==== */
	(qpw-assert (try
		(lambda () (begin
			(define unsupported-dj (qpir-dep-join true t-leaf pi-leaf (list) nil))
			(lower_to_scans_pass unsupported-dj) "no-error"))
		(lambda (e) "errored")) "errored"
		"lower errors on qpir-dep-join (unnest_pass must run first)")

	/* ==== End-to-end pipeline + lowering on correlated SUM ==== */
	(define e2e-inner (list "memcp-tests"
		(list (list "pi" "memcp-tests" "pi" false nil))
		(list (list "total"
			(list (quote aggregate) (mk-col "pi" "amount") (quote +) 0)))
		(list (quote equal??) (mk-col "pi" "k") (mk-col "po" "k"))
		(list) nil (list) nil nil))
	(define e2e-outer (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))
			(list "total" (list (quote inner_select) e2e-inner)))
		true (list) nil (list) nil nil))

	(define e2e-after-lift (lift_dep_joins_pass e2e-outer))
	(define e2e-after-unnest (unnest_pass e2e-after-lift))
	(define e2e-lowered (lower_to_scans_pass e2e-after-unnest))

	(qpw-assert (qpp-tuple? e2e-lowered) true
		"e2e: pipeline output is a 7-tuple ready for build_queryplan_inner")
	(qpw-assert (count (qpp-tuple-tables e2e-lowered)) 2
		"e2e: lowered 7-tuple has 2 tables (po + derived sq_N)")
	(qpw-assert (qpp-tuple? (nth (nth (qpp-tuple-tables e2e-lowered) 1) 2)) true
		"e2e: second table is a derived sub-7-tuple")
	/* The outer field "total" should reference sq_N.value (not pi.value) */
	(define total-field (nth (qpp-tuple-fields e2e-lowered) 1))
	(qpw-assert (nth total-field 0) "total" "e2e: outer field name preserved")
	(define total-expr (nth total-field 1))
	(qpw-assert (nth total-expr 0) (quote get_column)
		"e2e: outer total field references a column")

	(print "  qpw tests: " (- (qpw-tests "count") (qpw-tests "fail")) "/" (qpw-tests "count") " passed")
	(if (> (qpw-tests "fail") 0) (begin
		(print "")
		(print "  !!! queryplan-lower self-tests failed !!!")
		(print "  it is unsafe to run memcp in this configuration")
	) nil))
