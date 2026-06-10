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
== Tests for lib/queryplan-neumann.scm — public API ==

Covers:
- neumann_compile_select on hand-built parser-shaped 7-tuples
- End-to-end output is a clean 7-tuple (no inner_select markers)
- Errors-loudly on malformed inputs per FAQ §1
*/

(begin
	(print "testing queryplan-neumann (public API) ...")
	(define qpn-tests (newsession))
	(qpn-tests "count" 0)
	(qpn-tests "fail" 0)
	(define qpn-assert (lambda (val expected errormsg) (begin
		(qpn-tests "count" (+ (qpn-tests "count") 1))
		(if (equal? val expected)
			nil
			(begin
				(qpn-tests "fail" (+ (qpn-tests "fail") 1))
				(print "  qpn-test FAIL: " errormsg " (got: " val ", expected: " expected ")"))))))

	(define mk-col (lambda (tv col) (list (quote get_column) tv false col false)))

	/* ==== Trivial leaf-only query passes through cleanly ==== */
	(define t-trivial (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))) true
		(list) nil (list) nil nil))
	(define out-trivial (neumann_compile_select t-trivial))
	(qpn-assert (qpp-tuple? out-trivial) true "compile(trivial leaf) returns 7-tuple")
	(qpn-assert (qpp-tuple-condition out-trivial) true "compile(trivial leaf) preserves cond")
	(qpn-assert (qpp-tuple-tables out-trivial) (qpp-tuple-tables t-trivial) "tables preserved")

	/* Trivial query compiles end-to-end without error */
	(qpn-assert (try (lambda () (begin (neumann_compile_select t-trivial) "ok"))
		(lambda (e) "errored")) "ok"
		"compile(trivial) does not error")

	/* ==== Correlated scalar SUM goes through the full pipeline ==== */
	(define t-corr-inner (list "memcp-tests"
		(list (list "pi" "memcp-tests" "pi" false nil))
		(list (list "total"
			(list (quote aggregate) (mk-col "pi" "amount") (quote +) 0)))
		(list (quote equal??) (mk-col "pi" "k") (mk-col "po" "k"))
		(list) nil (list) nil nil))
	(define t-corr (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))
			(list "total" (list (quote inner_select) t-corr-inner)))
		true (list) nil (list) nil nil))

	(define out-corr (neumann_compile_select t-corr))
	(qpn-assert (qpp-tuple? out-corr) true "compile(correlated SUM) returns 7-tuple")
	(qpn-assert (count (qpp-tuple-tables out-corr)) 2
		"correlated SUM lowered: 2 tables (po + derived scalar helper)")

	/* Output 7-tuple must have NO inner_select markers anywhere */
	(define qpn-has-marker? (lambda (expr)
		(if (qpl-marker? expr) true
			(match expr
				(cons head args) (reduce (coalesceNil args (list)) (lambda (acc a)
					(or acc (qpn-has-marker? a))) false)
				false))))
	(define qpn-tuple-has-markers? (lambda (t) (begin
		(define fields-have (reduce (qpp-tuple-fields t) (lambda (acc pair) (match pair
			'(name expr) (or acc (qpn-has-marker? expr))
			acc)) false))
		(or fields-have
			(qpn-has-marker? (qpp-tuple-condition t))))))
	(qpn-assert (qpn-tuple-has-markers? out-corr) false
		"compile output has NO inner_select markers (lift removed them)")

	(qpn-assert (try (lambda () (begin (neumann_compile_select t-corr) "ok"))
		(lambda (e) "errored")) "ok"
		"compile(correlated SUM) does not error")

	/* ==== EXISTS via §11 COUNT rewrite goes through cleanly ==== */
	(define t-exists-inner (list "memcp-tests"
		(list (list "pi" "memcp-tests" "pi" false nil))
		(list (list "x" 1))
		(list (quote equal??) (mk-col "pi" "k") (mk-col "po" "k"))
		(list) nil (list) nil nil))
	(define t-exists (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id")))
		(list (quote inner_select_exists) t-exists-inner)
		(list) nil (list) nil nil))
	(define out-exists (neumann_compile_select t-exists))
	(qpn-assert (qpp-tuple? out-exists) true "compile(EXISTS) returns 7-tuple")
	(qpn-assert (qpn-tuple-has-markers? out-exists) false
		"EXISTS compile output has no markers")

	/* ==== Errors loudly on non-tuple input ==== */
	(qpn-assert (try
		(lambda () (begin (neumann_compile_select (list 1 2 3)) "no-error"))
		(lambda (e) "errored")) "errored"
		"compile errors on non-tuple input")

	/* ==== HAVING marker compiles through the no-fallback pipeline ==== */
	(define t-having (list "memcp-tests"
		(list (list "po" "memcp-tests" "po" false nil))
		(list (list "id" (mk-col "po" "id"))) true
		(list) (list (quote inner_select) t-corr-inner) (list) nil nil))
	(define out-having (neumann_compile_select t-having))
	(qpn-assert (qpp-tuple? out-having) true
		"compile(HAVING-marker tuple) returns 7-tuple")
	(qpn-assert (qpn-tuple-has-markers? out-having) false
		"compile(HAVING-marker tuple) output has no markers")

	(print "  qpn tests: " (- (qpn-tests "count") (qpn-tests "fail")) "/" (qpn-tests "count") " passed")
	(if (> (qpn-tests "fail") 0) (begin
		(print "")
		(print "  !!! queryplan-neumann self-tests failed !!!")
		(print "  it is unsafe to run memcp in this configuration")
	) nil))
