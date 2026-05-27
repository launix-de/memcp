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
== Tests for lib/queryplan-unnest.scm — Phase 1 ==

Covers:
  - Path-tracking walker (qpu-walk-with-path, qpu-lca-path)
  - Annotate helpers (qpu-collect-providers, qpu-collect-accessors,
    qpu-find-provider-path, qpu-find-enclosing-dep-join)
  - Tree rewrite primitive (qpu-rewrite-children, qpu-map-tree)
  - Trivial elimination — dep-join with no left-correlation becomes inner join
  - unnest_pass driver error path for non-trivial residuals (FAQ §1)
*/

(begin
	(print "testing queryplan-unnest ...")
	(define qpu-tests (newsession))
	(qpu-tests "count" 0)
	(qpu-tests "fail" 0)
	(define qpu-assert (lambda (val expected errormsg) (begin
		(qpu-tests "count" (+ (qpu-tests "count") 1))
		(if (equal? val expected)
			nil
			(begin
				(qpu-tests "fail" (+ (qpu-tests "fail") 1))
				(print "  qpu-test FAIL: " errormsg " (got: " val ", expected: " expected ")"))))))

	(define mk-col (lambda (tv col) (list (quote get_column) tv false col false)))
	(define mk-tuple (lambda (schema tables fields cond)
		(list schema tables fields cond '() nil '() nil nil)))

	/* ==== qpu-walk-with-path ==== */
	(define t-po (qpir-leaf (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))) true)))
	(define visited (newsession))
	(visited "n" 0)
	(qpu-walk-with-path t-po (lambda (node path)
		(visited "n" (+ (visited "n") 1))))
	(qpu-assert (visited "n") 1 "walker visits 1 node for a single leaf")

	(define t-select-leaf (qpir-select true t-po))
	(visited "n" 0)
	(qpu-walk-with-path t-select-leaf (lambda (node path)
		(visited "n" (+ (visited "n") 1))))
	(qpu-assert (visited "n") 2 "walker visits 2 nodes for select(leaf)")

	/* ==== qpu-lca-path ==== */
	(qpu-assert (count (qpu-lca-path '() '())) 0 "LCA of empty paths is empty")
	(qpu-assert (count (qpu-lca-path (list 1 2 3) (list 1 2 3))) 3 "LCA of equal paths is the full path")
	(qpu-assert (count (qpu-lca-path (list 1 2 3) (list 1 2 4))) 2 "LCA of diverging paths after 2 elements")
	(qpu-assert (count (qpu-lca-path (list 1 2) (list 1 2 3))) 2 "LCA when one is prefix of other")
	(qpu-assert (count (qpu-lca-path (list 1) (list 2))) 0 "LCA of disjoint paths is empty")

	/* ==== qpu-collect-providers / accessors ==== */
	(define providers (qpu-collect-providers t-po))
	(qpu-assert (count providers) 1 "single leaf has 1 provider entry")
	(qpu-assert (nth (nth providers 0) 1) (list "po") "leaf provides po alias")

	(define accessors (qpu-collect-accessors t-po))
	(qpu-assert (count accessors) 1 "single leaf has 1 accessor entry (its own field)")
	(qpu-assert (nth (nth accessors 0) 1) (list "po" "id") "accessor ref is (po id)")

	/* ==== qpu-find-provider-path ==== */
	(define found (qpu-find-provider-path (list "po" "id") providers))
	(qpu-assert (nil? found) false "find-provider-path finds existing alias")
	(define notfound (qpu-find-provider-path (list "qx" "k") providers))
	(qpu-assert (nil? notfound) true "find-provider-path returns nil for unknown alias")

	/* ==== qpu-trivial-eliminate ==== */
	/* Trivial dep-join: right side doesn't reference left's alias. Construct:
	   (dep-join true (leaf po) (leaf pi))  — pi-leaf's WHERE doesn't ref po. */
	(define t-pi-uncorr (qpir-leaf (mk-tuple "memcp-tests"
		(list (list "pi" "memcp-tests" "pi" false nil))
		(list (list "value" (mk-col "pi" "amount"))) true)))
	(define dj-trivial (qpir-dep-join true t-po t-pi-uncorr '() nil))
	(define trivial-elim (qpu-trivial-eliminate dj-trivial))
	(qpu-assert (qpir-kind trivial-elim) (quote qpir-join)
		"trivial dep-join (uncorrelated right) → qpir-join")
	(qpu-assert (qpir-join-type trivial-elim) (quote inner)
		"trivial elim produces inner join")

	/* Non-trivial dep-join: right's leaf has WHERE referencing po. */
	(define t-pi-corr (qpir-leaf (mk-tuple "memcp-tests"
		(list (list "pi" "memcp-tests" "pi" false nil))
		(list (list "value" (mk-col "pi" "amount")))
		(list (quote equal??) (mk-col "pi" "k") (mk-col "po" "k")))))
	(define dj-nontrivial (qpir-dep-join true t-po t-pi-corr '() nil))
	(define nontrivial-elim (qpu-trivial-eliminate dj-nontrivial))
	(qpu-assert (qpir-kind nontrivial-elim) (quote qpir-dep-join)
		"non-trivial dep-join stays as qpir-dep-join after trivial sub-pass")

	/* ==== qpu-map-tree ==== */
	(define identity-mapped (qpu-map-tree t-select-leaf (lambda (n) n)))
	(qpu-assert (qpir-kind identity-mapped) (quote qpir-select)
		"identity map preserves root kind")
	(qpu-assert (qpir-kind (qpir-select-child identity-mapped)) (quote qpir-leaf)
		"identity map preserves child kind")

	/* qpu-map-tree applied to a tree containing a trivial dep-join eliminates it */
	(define tree-with-trivial (qpir-select true dj-trivial))
	(define after-elim (qpu-trivial-eliminate-tree tree-with-trivial))
	(qpu-assert (qpir-kind after-elim) (quote qpir-select)
		"select wrapper preserved after trivial-eliminate-tree")
	(qpu-assert (qpir-kind (qpir-select-child after-elim)) (quote qpir-join)
		"inner dep-join eliminated within the tree")

	/* ==== qpu-count-dep-joins ==== */
	(qpu-assert (qpu-count-dep-joins t-po) 0 "leaf has 0 dep-joins")
	(qpu-assert (qpu-count-dep-joins dj-trivial) 1 "tree with one dep-join counts 1")
	(qpu-assert (qpu-count-dep-joins tree-with-trivial) 1 "nested dep-join also counts 1")
	(qpu-assert (qpu-count-dep-joins after-elim) 0 "after trivial-eliminate count = 0")

	/* ==== unnest_pass driver ==== */
	(define after-pass (unnest_pass tree-with-trivial))
	(qpu-assert (qpu-count-dep-joins after-pass) 0
		"unnest_pass on tree with only trivial dep-joins → 0 dep-joins remain")

	/* Non-trivial → error (Phase 2+ implementation required) */
	(define caught (try
		(lambda () (begin (unnest_pass dj-nontrivial) "no-error"))
		(lambda (e) "errored")))
	(qpu-assert caught "errored"
		"unnest_pass errors loudly on residual non-trivial dep-joins (FAQ §1)")

	(print "  qpu tests: " (- (qpu-tests "count") (qpu-tests "fail")) "/" (qpu-tests "count") " passed")
	(if (> (qpu-tests "fail") 0) (begin
		(print "")
		(print "  !!! queryplan-unnest self-tests failed !!!")
		(print "  it is unsafe to run memcp in this configuration")
	) nil))
