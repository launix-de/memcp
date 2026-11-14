/*
Copyright (C) 2023, 2024, 2026  Carl-Philip Haensch

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

/* Frontends register immutable callable values that may appear repeatedly in
emitted expressions. Keep their identity; names in the shared Scheme namespace
are not portable lexical bindings in compiled or distributed row callbacks. */
(define queryplan_callable_registry (coalesce queryplan_callable_registry (newsession)))
(queryplan_callable_registry "values" (coalesceNil (queryplan_callable_registry "values") '()))
(define register_queryplan_callables (lambda (values)
	(queryplan_callable_registry "values"
		(merge (list (queryplan_callable_registry "values") values)))))

(define queryplan_code_symbols (lambda (expr)
	(if (symbol? expr) (list expr)
		(match expr
			((symbol quote) _value) '()
			(cons head tail) (merge (map expr queryplan_code_symbols))
			_ '()))))
(define queryplan_callable_parameter (lambda (symbols index) (begin
	(define candidate (symbol (concat "__published_callables_" (string index))))
	(if (has? symbols candidate) (queryplan_callable_parameter symbols (+ index 1)) candidate))))
(define queryplan_publish_callable_expression (lambda (expr registered parameter bindings)
	(match expr
		((symbol quote) _value) expr
		(cons head tail) (begin
			/* Sharing must retain ordinary constant folding. Let the existing
			purity owner prove closed calls before turning their head into an
			unknown frame lookup; residual stateful calls remain executable. */
			(define constant (if (and (find registered (lambda (value) (identical? head value)) nil)
				(empty_list? (queryplan_code_symbols tail)))
				(try (lambda () (list true (tryfold expr))) (lambda (_not_constant) nil)) nil))
			(if constant (list (quote quote) (cadr constant))
				(map expr (lambda (item)
					(queryplan_publish_callable_expression item registered parameter bindings)))) )
		_ (begin
			(define callable (find registered (lambda (value) (identical? expr value)) nil))
			(if (nil? callable) expr (begin
				(define values (bindings "values"))
				(define index (coalesceNil (find (produceN (count values))
					(lambda (index) (identical? callable (nth values index))) nil) (count values)))
				(if (equal? index (count values)) (bindings "values" (append values callable)) nil)
				(list (quote nth) parameter index)))))))

/* Pass shared callables through an ordinary, explicit procedure frame. The
callee is compiled with an unknown vector parameter, so optimization cannot
substitute every callable body back into its consumers. apply preserves this
boundary while the actual immutable function vector remains visible once in
the emitted program, including EXPLAIN and procedure serialization. */
(define publish_queryplan_callables (lambda (code) (begin
	(define bindings (newsession))
	(bindings "values" '())
	(define parameter (queryplan_callable_parameter (queryplan_code_symbols code) 0))
	(define rewritten (queryplan_publish_callable_expression code
		(queryplan_callable_registry "values") parameter bindings))
	(define values (bindings "values"))
	(if (empty_list? values) code (begin
		(define runtime (list (quote session) (quote tx) (quote resultrow) (quote resultfields)))
		(define procedure (close_procedure (eval (list (quote lambda) (cons parameter runtime) rewritten))))
		(list (quote apply) procedure
			(cons (quote list) (cons (list (quote quote) values) runtime))))))))

/* Keep planner modules in pipeline order. Definitions are intentionally
loaded into the shared Scheme namespace for compatibility with existing
queryplan.scm importers. */
(import "queryplan-logical.scm")
(import "queryplan-iteration.scm")
(import "queryplan-local-relations.scm")
(import "queryplan-optimize.scm")
(import "queryplan-physical-scan.scm")
(import "queryplan-physical-expr.scm")
(import "queryplan-physical-plan.scm")
(import "queryplan-runtime-group.scm")
(import "queryplan-union-native.scm")
/* sql-types.scm defines sql_type_annotate_ir, called by optimize_logical_query
above. It is loaded last because its walk needs query-block/source helpers from
every module above; optimize_logical_query resolves the call late. */
(import "sql-types.scm")
