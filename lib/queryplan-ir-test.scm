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
== Tests for lib/queryplan-ir.scm ==

Snapshot-style tests proving every operator is constructible, kind detection
works, accessors return the right slots, child enumeration is correct, and
the free-variable F(N) computation matches BTW2025 §2.1 semantics.

Expression nodes follow the parser convention from lib/sql-parser.scm:
column refs are built as `(list (quote get_column) alias caseSens col caseSens)`.

Runs at server startup right after queryplan-ir.scm is loaded; prints a
pass/fail summary and a loud warning if any assertion fails.
*/

(begin
	(print "testing queryplan-ir ...")
	(define qpir-tests (newsession))
	(qpir-tests "count" 0)
	(qpir-tests "fail" 0)
	(define qpir-assert (lambda (val expected errormsg) (begin
		(qpir-tests "count" (+ (qpir-tests "count") 1))
		(if (equal? val expected)
			nil
			(begin
				(qpir-tests "fail" (+ (qpir-tests "fail") 1))
				(print "  qpir-test FAIL: " errormsg " (got: " val ", expected: " expected ")"))))))

	/* Helper: build a get_column ref list with bare-symbol head — matches both
	`(symbol get_column)` and `(quote get_column)` patterns in qpir-expr-column-refs. */
	(define qpir-mk-col (lambda (tv col)
		(list (quote get_column) tv false col false)))

	/* Helper: build (equal?? a b) as a data list */
	(define qpir-mk-eq (lambda (a b)
		(list (quote equal??) a b)))

	/* ==== Constructors + kind detection ==== */
	(define n-scan (qpir-scan "memcp-tests" "po"))
	(qpir-assert (qpir-kind n-scan) (quote qpir-scan) "qpir-scan kind")
	(qpir-assert (qpir-scan-schema n-scan) "memcp-tests" "qpir-scan schema accessor")
	(qpir-assert (qpir-scan-table n-scan) "po" "qpir-scan table accessor")

	(define n-select (qpir-select (qpir-mk-eq (qpir-mk-col "po" "k") 10) n-scan))
	(qpir-assert (qpir-kind n-select) (quote qpir-select) "qpir-select kind")
	(qpir-assert (qpir-select-child n-select) n-scan "qpir-select child accessor")

	(define n-map (qpir-map (list (list "id" (qpir-mk-col "po" "id"))) n-scan))
	(qpir-assert (qpir-kind n-map) (quote qpir-map) "qpir-map kind")

	(define n-groupby (qpir-groupby
		(list (qpir-mk-col "po" "k"))
		(list (list "total" (qpir-mk-col "po" "amount")))
		nil
		n-scan))
	(qpir-assert (qpir-kind n-groupby) (quote qpir-groupby) "qpir-groupby kind")

	(define n-window (qpir-window nil nil
		(list (list (quote limit) 10) (list (quote offset) 0))
		n-scan))
	(qpir-assert (qpir-kind n-window) (quote qpir-window) "qpir-window kind")

	(define n-rhs (qpir-scan "memcp-tests" "pi"))
	(define n-join (qpir-join (quote inner)
		(qpir-mk-eq (qpir-mk-col "po" "k") (qpir-mk-col "pi" "k"))
		n-scan n-rhs nil))
	(qpir-assert (qpir-kind n-join) (quote qpir-join) "qpir-join kind")
	(qpir-assert (qpir-join-type n-join) (quote inner) "qpir-join type")
	(qpir-assert (qpir-join-left n-join) n-scan "qpir-join left")
	(qpir-assert (qpir-join-right n-join) n-rhs "qpir-join right")

	(define n-dep (qpir-dep-join
		(qpir-mk-eq (qpir-mk-col "po" "k") (qpir-mk-col "pi" "k"))
		n-scan n-rhs '() nil))
	(qpir-assert (qpir-kind n-dep) (quote qpir-dep-join) "qpir-dep-join kind")

	(define n-union (qpir-union nil nil nil (list n-scan n-rhs)))
	(qpir-assert (qpir-kind n-union) (quote qpir-union) "qpir-union kind")
	(qpir-assert (count (qpir-union-branches n-union)) 2 "qpir-union branches")

	(define n-iter (qpir-iterate n-scan n-rhs '()))
	(qpir-assert (qpir-kind n-iter) (quote qpir-iterate) "qpir-iterate kind")

	(define n-leaf (qpir-leaf
		(list "memcp-tests" (list (list "po" "memcp-tests" "po" false nil))
			(list (list "id" (qpir-mk-col "po" "id")))
			true '() nil '() nil nil)))
	(qpir-assert (qpir-kind n-leaf) (quote qpir-leaf) "qpir-leaf kind")

	(qpir-assert (qpir-node? n-scan) true "qpir-node? scan")
	(qpir-assert (qpir-node? (list (quote other) (quote thing))) false "qpir-node? non-IR")
	(qpir-assert (qpir-node? "not a list") false "qpir-node? string")

	/* ==== Children enumeration ==== */
	(qpir-assert (count (qpir-children n-scan)) 0 "qpir-children scan = 0")
	(qpir-assert (count (qpir-children n-leaf)) 0 "qpir-children leaf = 0")
	(qpir-assert (count (qpir-children n-select)) 1 "qpir-children select = 1")
	(qpir-assert (count (qpir-children n-map)) 1 "qpir-children map = 1")
	(qpir-assert (count (qpir-children n-groupby)) 1 "qpir-children groupby = 1")
	(qpir-assert (count (qpir-children n-window)) 1 "qpir-children window = 1")
	(qpir-assert (count (qpir-children n-join)) 2 "qpir-children join = 2")
	(qpir-assert (count (qpir-children n-dep)) 2 "qpir-children dep-join = 2")
	(qpir-assert (count (qpir-children n-union)) 2 "qpir-children union = 2")
	(qpir-assert (count (qpir-children n-iter)) 2 "qpir-children iterate = 2")

	/* ==== Provided aliases ==== */
	(qpir-assert (qpir-provided-aliases n-scan) (list "po") "provided scan: po")
	(qpir-assert (qpir-provided-aliases n-rhs) (list "pi") "provided scan: pi")
	(define n-join-aliases (qpir-provided-aliases n-join))
	(qpir-assert (has? n-join-aliases "po") true "provided join contains po")
	(qpir-assert (has? n-join-aliases "pi") true "provided join contains pi")
	(qpir-assert (qpir-provided-aliases n-leaf) (list "po") "provided leaf: po")

	/* ==== Free-variable F(N) per BTW2025 §2.1 ==== */
	/* qpir-scan provides "po" and has no own refs → F = ∅ */
	(qpir-assert (count (qpir-free-vars n-scan)) 0 "F(scan) = empty")

	/* qpir-select on (po.k = 10) over po-scan: predicate refs po.k, child provides po → F = ∅ */
	(qpir-assert (count (qpir-free-vars n-select)) 0 "F(select po-only) = empty")

	/* qpir-select on outer ref (e.did = po.k) over po-scan: predicate refs po.k AND e.did,
	child provides only po → F = {(e did)} */
	(define n-correlated-select (qpir-select
		(qpir-mk-eq (qpir-mk-col "e" "did") (qpir-mk-col "po" "k"))
		n-scan))
	(define fv-correlated (qpir-free-vars n-correlated-select))
	(qpir-assert (count fv-correlated) 1 "F(select with outer e.did) has 1 free var")
	(qpir-assert (car fv-correlated) (list "e" "did") "F free var is (e did)")

	/* qpir-join over po and pi: condition refs both, children provide both → F = ∅ */
	(qpir-assert (count (qpir-free-vars n-join)) 0 "F(join po-join-pi) = empty")

	/* qpir-dep-join: same structure but type is qpir-dep-join → F still ∅ */
	(qpir-assert (count (qpir-free-vars n-dep)) 0 "F(dep-join po, pi) = empty")

	/* Correlated dep-join: select inside right correlates back to po.
	In dep-join as a whole: left provides po, right provides pi → po.k IS bound by left side. */
	(define n-pi-correlated-select (qpir-select
		(qpir-mk-eq (qpir-mk-col "po" "k") (qpir-mk-col "pi" "k"))
		(qpir-scan "memcp-tests" "pi")))
	(define n-dep-correlated (qpir-dep-join true n-scan n-pi-correlated-select '() nil))
	(qpir-assert (count (qpir-free-vars n-dep-correlated)) 0
		"F(dep-join with correlation po.k inside right) = empty (both aliases provided)")

	/* But isolating the right subtree alone shows the free var */
	(define fv-right-only (qpir-free-vars n-pi-correlated-select))
	(qpir-assert (count fv-right-only) 1 "F(right-of-correlated-dep-join) has 1 free var")
	(qpir-assert (car fv-right-only) (list "po" "k") "F free var is (po k)")

	/* ==== Pretty-printer doesn't crash ==== */
	(define show-out (qpir-show n-dep-correlated))
	(qpir-assert (> (strlen show-out) 0) true "qpir-show returns non-empty string")

	(print "  qpir tests: " (- (qpir-tests "count") (qpir-tests "fail")) "/" (qpir-tests "count") " passed")
	(if (> (qpir-tests "fail") 0) (begin
		(print "")
		(print "  !!! qpir IR self-tests failed !!!")
		(print "  it is unsafe to run memcp in this configuration")
	) nil))
