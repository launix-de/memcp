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
== Integration tests for the full BTW2025 pipeline ==

These tests exercise the L1 → L2 → L3 composition:
  parser-7-tuple  →
  alias_normalize_pass  →
  column_resolve_pass  →
  lift_dep_joins_pass  →
  unnest_pass  →
  qpir tree with NO dep-joins, F(root)=∅, ready for lower_to_scans

The input 7-tuples are HAND-CONSTRUCTED to match what the parser produces
(see lib/sql-parser.scm — sql_select_core returns the 9-element tuple).
This proves the passes COMPOSE correctly without requiring the SQL parser
to be wired in yet.

When lower_to_scans + parser-wiring land (Day 7 + Day 8 of the plan), the
SQL tests in tests/[0-9][0-9]_*.yaml become the integration suite. Until
then, these snapshot tests are the proof that the architecture works.
*/

(begin
	(print "testing queryplan-pipeline (integration) ...")
	(define qpipe-tests (newsession))
	(qpipe-tests "count" 0)
	(qpipe-tests "fail" 0)
	(define qpipe-assert (lambda (val expected errormsg) (begin
		(qpipe-tests "count" (+ (qpipe-tests "count") 1))
		(if (equal? val expected)
			nil
			(begin
				(qpipe-tests "fail" (+ (qpipe-tests "fail") 1))
				(print "  qpipe-test FAIL: " errormsg " (got: " val ", expected: " expected ")"))))))

	(define mk-col (lambda (tv col) (list (quote get_column) tv false col false)))

	/* ==== Test 1: SELECT po.id, (SELECT SUM(pi.amount) FROM pi WHERE pi.k=po.k) AS total FROM po ==== */
	/* The parser produces this 7-tuple for the above SQL. */
	(define inner-sum-tuple (list "memcp-tests"
		(list (list "pi" "memcp-tests" "pi" false nil))
		(list (list "total"
			(list (quote aggregate) (mk-col "pi" "amount") (quote +) 0)))
		(list (quote equal??) (mk-col "pi" "k") (mk-col "po" "k"))
		(list) nil (list) nil nil))
	(define outer-tuple (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))
			(list "total" (list (quote inner_select) inner-sum-tuple)))
		true (list) nil (list) nil nil))

	/* Run the full pipeline: */
	(define after-alias (alias_normalize_pass outer-tuple))
	(qpipe-assert (qpp-tuple? after-alias) true "alias_normalize_pass returns tuple")

	(define after-column (column_resolve_pass after-alias (list)))
	(qpipe-assert (qpp-tuple? after-column) true "column_resolve_pass returns tuple")

	(define after-lift (lift_dep_joins_pass after-column))
	(qpipe-assert (qpir-kind after-lift) (quote qpir-dep-join)
		"lift produces qpir-dep-join at root")
	(qpipe-assert (count (qpir-free-vars after-lift)) 0
		"after lift: F(root) = ∅ (dep-join.rhs-alias binds sq references)")

	(define after-unnest (unnest_pass after-lift))
	(qpipe-assert (qpir-kind after-unnest) (quote qpir-join)
		"after unnest: root is qpir-join (dep-join eliminated)")
	(qpipe-assert (qpir-join-type after-unnest) (quote inner)
		"after unnest: inner join")
	(qpipe-assert (count (qpir-free-vars after-unnest)) 0
		"after unnest: F(root) = ∅")

	/* Right side is qpir-groupby with the pushed po.k key */
	(define unnest-right (qpir-join-right after-unnest))
	(qpipe-assert (qpir-kind unnest-right) (quote qpir-groupby)
		"after unnest: right is qpir-groupby")
	(qpipe-assert (count (qpir-groupby-keys unnest-right)) 1
		"after unnest: groupby has 1 key (po.k pushed in via FAQ §33)")

	/* Join condition extracted from inner WHERE */
	(define join-pred (qpir-join-predicate after-unnest))
	(qpipe-assert (nth join-pred 0) (quote equal??)
		"after unnest: join predicate is the equality from inner WHERE")

	/* rhs-alias preserved through dep-join → join conversion. The exact sq_N
	   depends on the global counter (tests run in load order), so just check
	   the synthesized "sq_" prefix is present and that the alias is non-nil. */
	(define preserved-alias (qpir-join-rhs-alias after-unnest))
	(qpipe-assert (nil? preserved-alias) false
		"after unnest: rhs-alias is non-nil")
	(qpipe-assert (substr preserved-alias 0 3) "sq_"
		"after unnest: rhs-alias keeps sq_ prefix")

	/* ==== Test 2: SELECT po.id FROM po WHERE EXISTS (SELECT * FROM pi WHERE pi.k=po.k) ==== */
	/* EXISTS subquery: the parser emits inner_select_exists wrapping a 7-tuple.
	   FAQ §11: lift_dep_joins phase 3 rewrites it into a COUNT-based scalar via
	   COALESCE(COUNT(*), 0) > 0. */
	(define exists-inner-tuple (list "memcp-tests"
		(list (list "pi" "memcp-tests" "pi" false nil))
		(list (list "x" 1))   /* SELECT * → simplified to SELECT 1 for the test */
		(list (quote equal??) (mk-col "pi" "k") (mk-col "po" "k"))
		(list) nil (list) nil nil))
	(define exists-outer (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id")))
		(list (quote inner_select_exists) exists-inner-tuple)
		(list) nil (list) nil nil))

	(define ex-after-lift (lift_dep_joins_pass exists-outer))
	(qpipe-assert (qpir-kind ex-after-lift) (quote qpir-select)
		"EXISTS: after lift root is qpir-select (FAQ §11 COUNT > 0 boolean)")
	(qpipe-assert (count (qpir-free-vars ex-after-lift)) 0
		"EXISTS: after lift F(root) = ∅")

	(define ex-after-unnest (unnest_pass ex-after-lift))
	(qpipe-assert (count (qpir-free-vars ex-after-unnest)) 0
		"EXISTS: after unnest F(root) = ∅")
	(qpipe-assert (qpu-count-dep-joins ex-after-unnest) 0
		"EXISTS: after unnest no dep-joins remain")

	/* ==== Test 3: NO-correlation scalar subquery (uncorrelated) ==== */
	/* SELECT po.id, (SELECT SUM(pi.amount) FROM pi) FROM po — pi has no WHERE referencing po */
	(define uncorr-inner-tuple (list "memcp-tests"
		(list (list "pi" "memcp-tests" "pi" false nil))
		(list (list "total"
			(list (quote aggregate) (mk-col "pi" "amount") (quote +) 0)))
		true (list) nil (list) nil nil))
	(define uncorr-outer (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))
			(list "total" (list (quote inner_select) uncorr-inner-tuple)))
		true (list) nil (list) nil nil))

	(define uc-after-lift (lift_dep_joins_pass uncorr-outer))
	(define uc-after-unnest (unnest_pass uc-after-lift))
	(qpipe-assert (qpir-kind uc-after-unnest) (quote qpir-join)
		"uncorrelated SUM: after unnest root is qpir-join")
	(qpipe-assert (qpu-count-dep-joins uc-after-unnest) 0
		"uncorrelated SUM: no dep-joins after unnest")
	(qpipe-assert (count (qpir-free-vars uc-after-unnest)) 0
		"uncorrelated SUM: F(root) = ∅")
	/* For an uncorrelated subquery there are no outer-refs to push into the
	   groupby keys; the groupby stays as static-group (empty keys). */
	(define uc-right (qpir-join-right uc-after-unnest))
	(qpipe-assert (qpir-kind uc-right) (quote qpir-groupby)
		"uncorrelated SUM: right is qpir-groupby")
	(qpipe-assert (count (qpir-groupby-keys uc-right)) 0
		"uncorrelated SUM: groupby keys empty (no correlation)")

	/* ==== Test 4: nested correlated subqueries (recursive lift + bottom-up unnest) ==== */
	/* SQL: SELECT outer.id,
	          (SELECT (SELECT SUM(t1.x) FROM t1 WHERE t1.a=t0.a) AS inner
	           FROM t0 WHERE t0.b=outer.b) AS total
	        FROM outer
	   Two levels of correlated nesting. lift_dep_joins_pass recursively handles
	   the inner-most marker via wrap_inner_subquery, producing 2 qpir-dep-joins
	   stacked. unnest_pass bottom-up converts inner first then outer. */
	(define n-t1-sub (list "memcp-tests"
		(list (list "t1" "memcp-tests" "t1" false nil))
		(list (list "inner" (list (quote aggregate) (mk-col "t1" "x") (quote +) 0)))
		(list (quote equal??) (mk-col "t1" "a") (mk-col "t0" "a"))
		(list) nil (list) nil nil))
	(define n-t0-sub (list "memcp-tests"
		(list (list "t0" "memcp-tests" "t0" false nil))
		(list (list "inner" (list (quote inner_select) n-t1-sub)))
		(list (quote equal??) (mk-col "t0" "b") (mk-col "outer" "b"))
		(list) nil (list) nil nil))
	(define n-top (list "memcp-tests"
		(list (list "outer" "memcp-tests" "outer" false nil))
		(list (list "id" (mk-col "outer" "id"))
			(list "total" (list (quote inner_select) n-t0-sub)))
		true (list) nil (list) nil nil))

	(define n-after-lift (lift_dep_joins_pass n-top))
	(qpipe-assert (count (qpir-free-vars n-after-lift)) 0
		"nested-lift: F(root) = ∅")
	(qpipe-assert (qpu-count-dep-joins n-after-lift) 2
		"nested-lift: 2 dep-joins (outer + inner from recursive lift)")

	(define n-after-unnest (unnest_pass n-after-lift))
	(qpipe-assert (count (qpir-free-vars n-after-unnest)) 0
		"nested-unnest: F(root) = ∅")
	(qpipe-assert (qpu-count-dep-joins n-after-unnest) 0
		"nested-unnest: NO dep-joins remain (both converted)")

	(print "  qpipe tests: "
		(- (qpipe-tests "count") (qpipe-tests "fail")) "/" (qpipe-tests "count") " passed")
	(if (> (qpipe-tests "fail") 0) (begin
		(print "")
		(print "  !!! pipeline integration self-tests failed !!!")
		(print "  it is unsafe to run memcp in this configuration")
	) nil))
