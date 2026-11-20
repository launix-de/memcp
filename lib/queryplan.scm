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

/* Literal atoms cannot be immutable callable values. Most emitted leaves are
symbols or constants, so reject them before searching the callable registry. */
(define queryplan_registered_callable (lambda (value registered)
	(if (or (symbol? value) (or (string? value) (number? value))) nil
		(find registered (lambda (candidate) (identical? value candidate)) nil))))

/* Programs without registered values need no publication frame or symbol
catalog. Inspect the same executable positions as the publisher; quoted data
and opaque procedures keep their existing ownership. */
(define queryplan_uses_registered_callable? (lambda (expr registered)
	(match expr
		((symbol quote) _value) false
		(cons head tail) (or (queryplan_uses_registered_callable? head registered)
			(reduce tail (lambda (found item)
				(or found (queryplan_uses_registered_callable? item registered))) false))
		_ (not (nil? (queryplan_registered_callable expr registered))))))

(define queryplan_code_symbols (lambda (expr)
	(if (symbol? expr) (list expr)
		(match expr
			((symbol quote) _value) '()
			(cons head tail) (merge (map expr queryplan_code_symbols))
			_ '()))))
/* Folding admission needs only an existential answer. Avoid collecting every
symbol beneath each registered call; the full catalog above is still required
once when choosing a collision-free publication parameter. Quote opacity is
identical in both traversals. */
(define queryplan_code_has_symbol? (lambda (expr)
	(if (symbol? expr) true
		(match expr
			((symbol quote) _value) false
			(cons head tail) (not (nil? (find expr queryplan_code_has_symbol? nil)))
			_ false))))
(define queryplan_callable_parameter (lambda (symbols index) (begin
	(define candidate (symbol (concat "__published_callables_" (string index))))
	(if (has? symbols candidate) (queryplan_callable_parameter symbols (+ index 1)) candidate))))
(define queryplan_publish_callable_expression (lambda (expr registered parameter bindings)
	(match expr
		((symbol quote) _value) expr
		/* Logical/physical analysis has already consumed the binding adapter.
		Its actual closed identity has one operand, evaluated once; erase only
		that known value, leaving lexical same-spelled callees and quotes alone. */
		(cons head tail) (if (and (identical? head relational_iteration_parameter) (equal? (count tail) 1))
			(queryplan_publish_callable_expression (car tail) registered parameter bindings)
			(begin
				/* Sharing must retain ordinary constant folding. Let the existing
				purity owner prove closed calls before turning their head into an
				unknown frame lookup; residual stateful calls remain executable. */
				(define constant (if (and (queryplan_registered_callable head registered)
					(not (queryplan_code_has_symbol? tail)))
					(try (lambda () (list true (tryfold expr))) (lambda (_not_constant) nil)) nil))
				(if constant (list (quote quote) (cadr constant))
					(map expr (lambda (item)
						(queryplan_publish_callable_expression item registered parameter bindings))))))
		_ (begin
			(define callable (queryplan_registered_callable expr registered))
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
(define publish_queryplan_callables (lambda (code invocation_inputs) (begin
	(define registered (queryplan_callable_registry "values"))
	(if (not (queryplan_uses_registered_callable? code registered)) code (begin
		(define bindings (newsession))
		(bindings "values" '())
		(define parameter (queryplan_callable_parameter
			(merge (list (queryplan_code_symbols code) invocation_inputs)) 0))
		(define rewritten (queryplan_publish_callable_expression code
			registered parameter bindings))
		(define values (bindings "values"))
		(if (empty_list? values) code (begin
			/* Persistent row callbacks must not gain unused request inputs merely
			because a callable is shared. Analyze the original complete program:
			its actual callees still have their ordinary invocation identities,
			before publication turns them into binding-vector lookup heads. */
			(define runtime (expression_referenced_bindings code
				(merge_unique (list
					(list (quote session) (quote tx) (quote resultrow) (quote resultfields))
					invocation_inputs))))
			(define procedure (close_procedure (eval (list (quote lambda) (cons parameter runtime) rewritten))))
			(list (quote apply) procedure
				(cons (quote list) (cons (list (quote quote) values) runtime))))))))))

/* Keep planner modules in pipeline order. Definitions are intentionally
loaded into the shared Scheme namespace for compatibility with existing
queryplan.scm importers. */
(import "queryplan-logical.scm")
(import "queryplan-iteration.scm")
(import "queryplan-local-relations.scm")
(import "queryplan-optimize.scm")
(import "queryplan-physical-scan.scm")
(import "queryplan-result-consumer.scm")
(import "queryplan-physical-expr.scm")
(import "queryplan-physical-plan.scm")
(import "queryplan-runtime-group.scm")
(import "queryplan-union-native.scm")
/* sql-types.scm defines sql_type_annotate_ir, called by optimize_logical_query
above. It is loaded last because its walk needs query-block/source helpers from
every module above; optimize_logical_query resolves the call late. */
(import "sql-types.scm")
