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
== Layer 2 — lift_dep_joins_pass (7-tuple → L1 operator IR) ==

This pass transforms a 7-tuple (normalized by alias_normalize_pass and
column_resolve_pass) into a Layer-1 IR tree. Every `inner_select`,
`inner_select_in`, `inner_select_exists` marker in the tuple's expression
slots becomes an explicit `qpir-dep-join` node in the IR — making
correlations FIRST-CLASS so the holistic unnesting pass (Layer 3) can
eliminate them top-down per BTW2025 §3.

Invariant after this pass:
  No leaf 7-tuple's fields/condition/group/having/order contains any
  inner_select* marker. All correlations live at the operator level.

Marker shapes emitted by the parser (see lib/sql-parser.scm):
  (inner_select        sub-7tuple)     — scalar SELECT subquery
  (inner_select_in     a sub-7tuple)   — `a IN (SELECT …)`
  (inner_select_exists sub-7tuple)     — EXISTS (SELECT …)

Implementation strategy (incremental, per FAQ §1 — NO fallback plans):
  Phase 1 (this commit):
    - Detect markers in a tuple's expression slots.
    - No markers → return (qpir-leaf tuple).
    - Single scalar inner_select in SELECT-list field → lift into
      (qpir-map projections (qpir-dep-join true outer-leaf inner-leaf '()))
      where the marker is replaced by a get_column reference to a fresh
      sq_alias.value column, and the inner subquery becomes its own leaf.
    - Anything else (markers in WHERE/HAVING, IN/EXISTS, multiple markers,
      UNION ALL inner, recursive) → PANIC with a descriptive message.
      Subsequent phases implement these cases incrementally.

  Later phases handle: multiple markers in same field, WHERE-level markers,
  HAVING-level markers (split groupby/select), IN/EXISTS markers,
  multi-table outer 7-tuples, UNION ALL inner subqueries, recursive CTEs.

Per FAQ §15: when a phase encounters an unhandled shape it MUST panic,
never silently degrade to legacy code paths. The panic message names the
shape so the next phase implementer knows exactly what to add.
*/

/* ==================== Marker detection ==================== */

/* qpl-marker-kind — return the inner_select kind symbol of expr, or nil.
Returns: (quote inner_select), (quote inner_select_in), (quote inner_select_exists), or nil. */
(define qpl-marker-kind (lambda (expr) (match expr
	(cons sym _) (match sym
		(symbol inner_select)         (quote inner_select)
		'(quote inner_select)         (quote inner_select)
		'inner_select                 (quote inner_select)
		(symbol inner_select_in)      (quote inner_select_in)
		'(quote inner_select_in)      (quote inner_select_in)
		'inner_select_in              (quote inner_select_in)
		(symbol inner_select_exists)  (quote inner_select_exists)
		'(quote inner_select_exists)  (quote inner_select_exists)
		'inner_select_exists          (quote inner_select_exists)
		nil)
	nil)))

(define qpl-marker? (lambda (expr) (not (nil? (qpl-marker-kind expr)))))

/* qpl-collect-markers — walk an expression tree and return a list of every
marker subexpression encountered (in left-to-right order, depth-first). */
(define qpl-collect-markers (lambda (expr)
	(if (qpl-marker? expr)
		(list expr)
		(match expr
			(cons _sym args) (reduce (coalesceNil args '()) (lambda (acc a)
				(merge acc (qpl-collect-markers a))) '())
			'()))))

/* qpl-marker-subquery — extract the sub-7tuple from a marker. */
(define qpl-marker-subquery (lambda (marker)
	(match (qpl-marker-kind marker)
		(quote inner_select)         (nth marker 1)
		(quote inner_select_in)      (nth marker 2)
		(quote inner_select_exists)  (nth marker 1)
		nil)))

/* qpl-marker-lhs — for inner_select_in, return the LHS expression `a` in
`a IN (SELECT …)`. For other marker kinds, returns nil. */
(define qpl-marker-lhs (lambda (marker)
	(match (qpl-marker-kind marker)
		(quote inner_select_in) (nth marker 1)
		nil)))

/* qpl-tuple-has-markers? — true if any expression slot of the tuple contains
at least one inner_select* marker. */
(define qpl-tuple-has-markers? (lambda (t) (begin
	(define collected (newsession))
	(collected "n" 0)
	(qpp-apply-to-tuple t (lambda (e) (begin
		(collected "n" (+ (collected "n") (count (qpl-collect-markers e))))
		e)))
	(> (collected "n") 0))))

/* ==================== Lift driver ==================== */

/* qpl-fresh-sq-alias — generate a unique sq_alias name. Uses a session
counter so distinct lifts within one compile share no name. */
(define qpl-sq-counter (newsession))
(qpl-sq-counter "n" 0)
(define qpl-fresh-sq-alias (lambda () (begin
	(qpl-sq-counter "n" (+ (qpl-sq-counter "n") 1))
	(concat "sq_" (string (qpl-sq-counter "n"))))))

/* qpl-marker-is-scalar-list-field? — true when exactly ONE field's expression
is a bare scalar inner_select marker (no other markers anywhere else in the
tuple). This is the shape Phase 1 handles directly. */
(define qpl-marker-is-scalar-list-field? (lambda (t) (begin
	(define fields (qpp-tuple-fields t))
	(define marker-fields (filter fields (lambda (pair) (match pair
		'(_name expr) (equal? (qpl-marker-kind expr) (quote inner_select))
		false))))
	(define non-field-markers (count
		(merge
			(qpl-collect-markers (qpp-tuple-condition t))
			(reduce (coalesceNil (qpp-tuple-group t) '()) (lambda (acc e)
				(merge acc (qpl-collect-markers e))) '())
			(qpl-collect-markers (qpp-tuple-having t))
			(reduce (coalesceNil (qpp-tuple-order t) '()) (lambda (acc item) (match item
				'(expr _dir) (merge acc (qpl-collect-markers expr))
				acc)) '()))))
	(and (equal? (count marker-fields) 1) (equal? non-field-markers 0)))))

/* qpl-replace-marker-with-col — replace the scalar inner_select in a field's
expression with a get_column reference to sq_alias.value. */
(define qpl-replace-marker-with-col (lambda (fields sq-alias)
	(map fields (lambda (pair) (match pair
		'(name expr) (if (equal? (qpl-marker-kind expr) (quote inner_select))
			(list name (list (quote get_column) sq-alias false "value" false))
			pair)
		pair)))))

/* qpl-extract-marker-subquery-from-fields — find the single scalar marker in
fields and return its inner subquery 7-tuple. */
(define qpl-extract-marker-subquery-from-fields (lambda (fields)
	(reduce fields (lambda (found pair) (match pair
		'(_name expr) (if (and (nil? found) (equal? (qpl-marker-kind expr) (quote inner_select)))
			(qpl-marker-subquery expr)
			found)
		found))
		nil)))

/* qpl-wrap-inner-subquery-as-leaf — convert a parser-emitted inner subquery
7-tuple into a (qpir-leaf …). The subquery is renamed so its single output
column is `value` so the outer can reference (get_column sq_alias value).
For Phase 1 we require the subquery's fields list to be a single
(<original_name> expr); we rename that to ("value" expr). */
(define qpl-wrap-inner-subquery-as-leaf (lambda (sub)
	(if (not (qpp-tuple? sub))
		(error "qpl-wrap-inner-subquery-as-leaf: sub is not a 7-tuple")
		(begin
			(define fields (qpp-tuple-fields sub))
			(if (equal? (count fields) 1)
				(qpir-leaf (qpp-rebuild-tuple
					(qpp-tuple-schema sub)
					(qpp-tuple-tables sub)
					(list (list "value" (nth (nth fields 0) 1)))
					(qpp-tuple-condition sub)
					(qpp-tuple-group sub)
					(qpp-tuple-having sub)
					(qpp-tuple-order sub)
					(qpp-tuple-limit sub)
					(qpp-tuple-offset sub)))
				(error "qpl-wrap-inner-subquery-as-leaf: Phase 1 requires single-field inner subquery"))))))

/* lift_dep_joins_pass — the L2 → L1 transformation.
Input: a 7-tuple (post-alias-normalize and column-resolve).
Output: an L1 IR tree (qpir-* node) with NO inner_select markers in any leaf. */
(define lift_dep_joins_pass (lambda (t)
	(if (not (qpp-tuple? t))
		(error "lift_dep_joins_pass: input is not a 7-tuple")
		(if (not (qpl-tuple-has-markers? t))
			/* No markers: the tuple is already a pure SQL block. */
			(qpir-leaf t)
			/* Has markers: dispatch by shape. */
			(if (qpl-marker-is-scalar-list-field? t)
				(qpl-lift-scalar-list-field t)
				(error (concat "lift_dep_joins_pass Phase 1: unhandled marker shape. "
					"Supported: no markers, OR single scalar inner_select in SELECT-list field. "
					"Found markers elsewhere — Phase 2+ will add WHERE/HAVING/IN/EXISTS handling.")))))))

/* qpl-lift-scalar-list-field — Phase 1 lifter for the supported shape. */
(define qpl-lift-scalar-list-field (lambda (t) (begin
	(define sq-alias (qpl-fresh-sq-alias))
	(define inner-sub (qpl-extract-marker-subquery-from-fields (qpp-tuple-fields t)))
	(define outer-leaf-fields (qpl-replace-marker-with-col (qpp-tuple-fields t) sq-alias))
	(define outer-leaf-tuple (qpp-rebuild-tuple
		(qpp-tuple-schema t)
		(qpp-tuple-tables t)
		outer-leaf-fields
		(qpp-tuple-condition t)
		(qpp-tuple-group t)
		(qpp-tuple-having t)
		(qpp-tuple-order t)
		(qpp-tuple-limit t)
		(qpp-tuple-offset t)))
	(define inner-leaf (qpl-wrap-inner-subquery-as-leaf inner-sub))
	(qpir-dep-join true (qpir-leaf outer-leaf-tuple) inner-leaf '()))))
