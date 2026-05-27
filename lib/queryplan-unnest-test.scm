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
		(list schema tables fields cond (list) nil (list) nil nil)))

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
	(qpu-assert (count (qpu-lca-path (list) (list))) 0 "LCA of empty paths is empty")
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
	(define dj-trivial (qpir-dep-join true t-po t-pi-uncorr (list) nil))
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
	(define dj-nontrivial (qpir-dep-join true t-po t-pi-corr (list) nil))
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

	/* ==== annotate_dep_joins public API ==== */
	(define ann-trivial (annotate_dep_joins dj-trivial))
	(qpu-assert (count (ann-trivial "map")) 0
		"annotate: trivial dep-join has 0 accessor entries")
	(define ann-nontrivial (annotate_dep_joins dj-nontrivial))
	(qpu-assert (> (count (ann-nontrivial "map")) 0) true
		"annotate: non-trivial dep-join has at least 1 accessor entry")

	/* ==== qpu-accessing-of ==== */
	(define ann-acc-list (qpu-accessing-of ann-nontrivial dj-nontrivial))
	(qpu-assert (> (count ann-acc-list) 0) true
		"qpu-accessing-of returns at least one accessor path for the dep-join")

	/* ==== qpu-collect-outer-refs ==== */
	(define outer-refs-trivial (qpu-collect-outer-refs dj-trivial))
	(qpu-assert (count outer-refs-trivial) 0
		"collect-outer-refs: trivial dep-join has 0 outer refs")
	(define outer-refs-nontrivial (qpu-collect-outer-refs dj-nontrivial))
	(qpu-assert (count outer-refs-nontrivial) 1
		"collect-outer-refs: non-trivial dep-join (with WHERE pi.k=po.k) has 1 outer ref")
	(qpu-assert (nth (nth outer-refs-nontrivial 0) 1) "po"
		"collect-outer-refs: outer ref tblvar is po")
	(qpu-assert (nth (nth outer-refs-nontrivial 0) 3) "k"
		"collect-outer-refs: outer ref col is k")

	/* ==== qpu-push-outer-refs-into-groupby ==== */
	(define gb-bare (qpir-groupby (list)
		(list (list "value" (list (quote aggregate) (mk-col "pi" "amount") (quote +) 0)))
		nil (qpir-leaf (mk-tuple "memcp-tests"
			(list (list "pi" "memcp-tests" "pi" false nil))
			(list (list "amount" (mk-col "pi" "amount")))
			true))))
	(qpu-assert (count (qpir-groupby-keys gb-bare)) 0 "fresh groupby: 0 keys")
	(define gb-pushed (qpu-push-outer-refs-into-groupby gb-bare
		(list (mk-col "po" "k"))))
	(qpu-assert (count (qpir-groupby-keys gb-pushed)) 1
		"after push-outer-refs: groupby has 1 key")
	(qpu-assert (nth (nth (qpir-groupby-keys gb-pushed) 0) 1) "po"
		"pushed key's tblvar is po")
	(qpu-assert (nth (nth (qpir-groupby-keys gb-pushed) 0) 3) "k"
		"pushed key's col is k")
	(qpu-assert (count (qpir-groupby-aggs gb-pushed)) 1
		"after push: aggs preserved (1 entry)")
	(qpu-assert (qpir-groupby-having gb-pushed) nil
		"after push: having preserved (nil)")

	/* ==== Predicate splitting helpers ==== */
	(qpu-assert (count (qpu-and-conjuncts true)) 0 "and-conjuncts of true is empty")
	(qpu-assert (count (qpu-and-conjuncts (list (quote equal??) 1 2))) 1
		"and-conjuncts of bare predicate is 1")
	(qpu-assert (count (qpu-and-conjuncts
		(list (quote and) (list (quote equal??) 1 2) (list (quote >) 3 4)))) 2
		"and-conjuncts splits (and a b) into 2")
	(qpu-assert (qpu-and-from-conjuncts (list)) true
		"and-from-conjuncts of empty is true")
	(qpu-assert (qpu-and-from-conjuncts (list (list (quote >) 1 2))) (list (quote >) 1 2)
		"and-from-conjuncts of 1 returns bare predicate")

	(qpu-assert (qpu-expr-references-aliases? (mk-col "po" "k") (list "po")) true
		"expr-refs-aliases positive case")
	(qpu-assert (qpu-expr-references-aliases? (mk-col "pi" "k") (list "po")) false
		"expr-refs-aliases negative case")

	/* split: predicate fully outer-correlated → entire predicate to outer */
	(define split-corr (qpu-split-predicate
		(list (quote equal??) (mk-col "pi" "k") (mk-col "po" "k"))
		(list "po")))
	(qpu-assert (nth split-corr 0) (list (quote equal??) (mk-col "pi" "k") (mk-col "po" "k"))
		"split: pure outer-corr predicate → outer side")
	(qpu-assert (nth split-corr 1) true "split: pure outer-corr → pure side is true")

	/* split: mixed AND → conjuncts go to respective sides */
	(define split-mixed (qpu-split-predicate
		(list (quote and)
			(list (quote equal??) (mk-col "pi" "k") (mk-col "po" "k"))
			(list (quote >) (mk-col "pi" "amount") 100))
		(list "po")))
	(qpu-assert (nth split-mixed 0) (list (quote equal??) (mk-col "pi" "k") (mk-col "po" "k"))
		"split mixed: outer side = corr conjunct")
	(qpu-assert (nth split-mixed 1) (list (quote >) (mk-col "pi" "amount") 100)
		"split mixed: pure side = inner conjunct")

	/* ==== qpu-unnest-dep-join — end-to-end correlated SUM shape ==== */
	(define inner-leaf (qpir-leaf (mk-tuple "memcp-tests"
		(list (list "pi" "memcp-tests" "pi" false nil))
		(list (list "amount" (mk-col "pi" "amount")) (list "k" (mk-col "pi" "k")))
		true)))
	(define inner-select (qpir-select
		(list (quote equal??) (mk-col "pi" "k") (mk-col "po" "k"))
		inner-leaf))
	(define inner-gb (qpir-groupby (list)
		(list (list "value"
			(list (quote aggregate) (mk-col "pi" "amount") (quote +) 0)))
		nil inner-select))
	(define outer-leaf-su (qpir-leaf (mk-tuple "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))
			(list "total" (list (quote get_column) "sq_1" false "value" false)))
		true)))
	(define dj-su (qpir-dep-join true outer-leaf-su inner-gb (list) "sq_1"))
	(qpu-assert (count (qpir-free-vars dj-su)) 0
		"input dj-su has F = ∅ (rhs-alias makes sq_1 bound)")
	(qpu-assert (count (qpu-collect-outer-refs dj-su)) 1
		"dj-su has 1 outer ref (po.k)")

	(define unnested-su (qpu-unnest-dep-join dj-su))
	(qpu-assert (qpir-kind unnested-su) (quote qpir-join)
		"unnested dj-su is qpir-join")
	(qpu-assert (qpir-join-type unnested-su) (quote inner)
		"unnested dj-su is inner join")
	(qpu-assert (qpir-join-rhs-alias unnested-su) "sq_1"
		"unnested join preserves rhs-alias = sq_1")
	(qpu-assert (qpir-join-left unnested-su) outer-leaf-su
		"unnested join left = original outer")

	/* The right is qpir-groupby with pi.k now in keys, child is qpir-leaf (select dropped). */
	(define unnested-right (qpir-join-right unnested-su))
	(qpu-assert (qpir-kind unnested-right) (quote qpir-groupby)
		"unnested right is qpir-groupby")
	(qpu-assert (count (qpir-groupby-keys unnested-right)) 1
		"groupby has 1 key after outer-refs push (was 0)")
	/* After phase 3 cclasses substitution: po.k gets substituted to pi.k since
	   the inner WHERE pi.k=po.k put them in the same equivalence class. So the
	   pushed groupby key references pi.k (the inner-side equivalent), not po.k.
	   This is the architecturally correct behavior — the groupby's child (the
	   leaf) provides pi but NOT po, so a key referencing po would be unbound. */
	(qpu-assert (nth (nth (qpir-groupby-keys unnested-right) 0) 1) "pi"
		"groupby key references pi (cclasses substituted po → pi per FAQ §41)")
	(qpu-assert (count (qpir-groupby-aggs unnested-right)) 1
		"groupby aggs preserved")
	(qpu-assert (qpir-kind (qpir-groupby-child unnested-right)) (quote qpir-leaf)
		"select was extracted (groupby child is now the inner leaf)")

	/* Join's predicate: cclasses build IS NOT DISTINCT FROM conjunct linking
	   po.k to pi.k (per FAQ §41 — NULL-safe equality). */
	(define unnested-pred (qpir-join-predicate unnested-su))
	(qpu-assert (nth unnested-pred 0) (quote equal??)
		"join predicate is equal?? (IS NOT DISTINCT FROM from cclasses)")

	/* F(root) of the unnested tree must be ∅ */
	(qpu-assert (count (qpir-free-vars unnested-su)) 0
		"F(unnested tree) = ∅ — fully unnested")

	(qpu-assert (qpu-count-dep-joins unnested-su) 0
		"no qpir-dep-join remains after unnest")

	/* ==== unnest_pass driver — end-to-end ==== */
	(define pass-out (unnest_pass dj-su))
	(qpu-assert (qpu-count-dep-joins pass-out) 0
		"unnest_pass on correlated SUM eliminates all dep-joins")
	(qpu-assert (qpir-kind pass-out) (quote qpir-join)
		"unnest_pass result is qpir-join for the canonical SUM shape")

	(print "  qpu tests: " (- (qpu-tests "count") (qpu-tests "fail")) "/" (qpu-tests "count") " passed")
	(if (> (qpu-tests "fail") 0) (begin
		(print "")
		(print "  !!! queryplan-unnest self-tests failed !!!")
		(print "  it is unsafe to run memcp in this configuration")
	) nil))
