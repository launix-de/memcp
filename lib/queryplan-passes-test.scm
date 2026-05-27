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
== Tests for lib/queryplan-passes.scm ==

Snapshot tests for the L2 pre-passes. Each test feeds a hand-built 7-tuple
through one pass and asserts the result matches the expected normalization.

Runs at server startup right after queryplan-passes.scm is loaded.
*/

(begin
	(print "testing queryplan-passes ...")
	(define qpp-tests (newsession))
	(qpp-tests "count" 0)
	(qpp-tests "fail" 0)
	(define qpp-assert (lambda (val expected errormsg) (begin
		(qpp-tests "count" (+ (qpp-tests "count") 1))
		(if (equal? val expected)
			nil
			(begin
				(qpp-tests "fail" (+ (qpp-tests "fail") 1))
				(print "  qpp-test FAIL: " errormsg " (got: " val ", expected: " expected ")"))))))

	/* Helpers — build expressions the same way the parser does */
	(define mk-col (lambda (tv col)
		(list (quote get_column) tv false col false)))
	(define mk-col-ti (lambda (tv col ti ci)
		(list (quote get_column) tv ti col ci)))

	/* ==== qpp-tuple? shape detection ==== */
	(define tuple-ok (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id")))
		true '() nil '() nil nil))
	(qpp-assert (qpp-tuple? tuple-ok) true "qpp-tuple? accepts 9-element list")
	(qpp-assert (qpp-tuple? (list 1 2 3)) false "qpp-tuple? rejects 3-element list")
	(qpp-assert (qpp-tuple? nil) false "qpp-tuple? rejects nil")
	(qpp-assert (qpp-tuple? "string") false "qpp-tuple? rejects string")
	(qpp-assert (qpp-tuple-schema tuple-ok) "memcp-tests" "qpp-tuple-schema accessor")
	(qpp-assert (qpp-tuple-condition tuple-ok) true "qpp-tuple-condition accessor")

	/* ==== alias_normalize_pass ==== */
	/* Simple alias — no rewrite needed */
	(define t-simple (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id")))
		(list (quote equal??) (mk-col "po" "k") 10)
		'() nil '() nil nil))
	(define t-simple-norm (alias_normalize_pass t-simple))
	(qpp-assert (qpp-tuple? t-simple-norm) true "alias_normalize_pass returns 9-tuple")
	(qpp-assert (qpp-tuple-schema t-simple-norm) "memcp-tests" "alias_normalize preserves schema")
	(qpp-assert (qpp-tuple-tables t-simple-norm) (qpp-tuple-tables t-simple) "alias_normalize preserves tables")
	(qpp-assert (qpp-tuple-fields t-simple-norm) (qpp-tuple-fields t-simple) "alias_normalize leaves simple aliases unchanged")
	(qpp-assert (qpp-tuple-condition t-simple-norm) (qpp-tuple-condition t-simple) "alias_normalize leaves simple condition unchanged")

	/* (visible_alias canonical) provenance — visible_occurrence_alias picks visible side */
	(define t-vis (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "k" (mk-col (list (quote visible_alias) "canonical_po") "k")))
		true '() nil '() nil nil))
	(define t-vis-norm (alias_normalize_pass t-vis))
	(qpp-assert (qpp-tuple-fields t-vis-norm)
		(list (list "k" (mk-col (quote visible_alias) "k")))
		"alias_normalize picks visible side of (visible_alias canonical) provenance")

	/* alias_normalize is idempotent — running twice gives the same result */
	(qpp-assert (alias_normalize_pass t-simple-norm) t-simple-norm
		"alias_normalize_pass is idempotent")

	/* ==== column_resolve_pass ==== */
	/* Tested with empty schemas — when no schema exists for an alias, the resolver
	   leaves the (get_column) expression alone (legacy contract: unresolved refs
	   pass through to replace_find_column for late repair). This still proves the
	   pass walks every slot and rebuilds the tuple shape. Real-schema integration
	   is covered by the existing tests/*.yaml suite via build_queryplan_inner. */
	(define empty-schemas '())

	/* already-canonical (ti=ci=false) — pass-through regardless of schemas */
	(define t-can (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col-ti "po" "id" false false)))
		(mk-col-ti "po" "k" false false)
		'() nil '() nil nil))
	(define t-can-res (column_resolve_pass t-can empty-schemas))
	(qpp-assert (qpp-tuple? t-can-res) true "column_resolve_pass returns 9-tuple")
	(qpp-assert (qpp-tuple-fields t-can-res) (qpp-tuple-fields t-can)
		"column_resolve leaves already-canonical expressions unchanged")
	(qpp-assert (qpp-tuple-condition t-can-res) (qpp-tuple-condition t-can)
		"column_resolve leaves already-canonical condition unchanged")

	/* ti=true with empty schemas — resolver returns input (no schema to resolve against) */
	(define t-ti (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col-ti "po" "id" true true)))
		true '() nil '() nil nil))
	(define t-ti-res (column_resolve_pass t-ti empty-schemas))
	(qpp-assert (qpp-tuple? t-ti-res) true "column_resolve_pass returns 9-tuple for ti=true")
	(qpp-assert (count (qpp-tuple-fields t-ti-res)) 1 "column_resolve preserves field count")

	/* column_resolve_pass is idempotent on canonical input */
	(qpp-assert (column_resolve_pass t-can-res empty-schemas) t-can-res
		"column_resolve_pass is idempotent on canonical input")

	/* ==== qpp-apply-to-tuple — generic walker exercises every slot ==== */
	(define t-walk (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))
			(list "k" (mk-col "po" "k")))
		(list (quote equal??) (mk-col "po" "k") 10)
		(list (mk-col "po" "k"))
		(list (quote equal??) (mk-col "po" "id") 1)
		(list (list (mk-col "po" "id") (quote asc)))
		100 0))
	(define count-cells (newsession))
	(count-cells "n" 0)
	(define count-fn (lambda (expr) (begin
		(count-cells "n" (+ (count-cells "n") 1))
		expr)))
	(define t-walk-out (qpp-apply-to-tuple t-walk count-fn))
	/* Walk should hit: 2 field exprs + condition + 1 group + having + 1 order expr = 6 */
	(qpp-assert (count-cells "n") 6 "qpp-apply-to-tuple visits each expression slot")
	(qpp-assert (qpp-tuple-limit t-walk-out) 100 "qpp-apply-to-tuple preserves limit")
	(qpp-assert (qpp-tuple-offset t-walk-out) 0 "qpp-apply-to-tuple preserves offset")
	(qpp-assert (qpp-tuple-schema t-walk-out) "memcp-tests" "qpp-apply-to-tuple preserves schema")

	(print "  qpp tests: " (- (qpp-tests "count") (qpp-tests "fail")) "/" (qpp-tests "count") " passed")
	(if (> (qpp-tests "fail") 0) (begin
		(print "")
		(print "  !!! queryplan-passes self-tests failed !!!")
		(print "  it is unsafe to run memcp in this configuration")
	) nil))
