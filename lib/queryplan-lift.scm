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

Transforms a normalized 7-tuple (post alias/column normalization) into a
Layer-1 IR tree. Every `inner_select`, `inner_select_in`,
`inner_select_exists` marker in the tuple's expression slots becomes an
explicit `qpir-dep-join` node so the holistic unnesting pass (Day 4-5)
can eliminate correlations top-down per BTW2025 §3.

Invariant after this pass:
  No qpir-leaf's 7-tuple contains any inner_select* marker — every
  correlation has been lifted into the operator level.

Marker shapes from the parser (lib/sql-parser.scm):
  (inner_select        sub-7tuple)     — scalar SELECT subquery
  (inner_select_in     a sub-7tuple)   — `a IN (SELECT …)`
  (inner_select_exists sub-7tuple)     — EXISTS (SELECT …)

Algorithm (current scope — Phase 2):

  1. If the tuple has no markers anywhere, return (qpir-leaf tuple).

  2. Collect every `inner_select` (scalar) marker in fields and condition
     slots. For each marker, allocate a fresh sq_N alias. Substitute the
     marker in-place with (get_column sq_N false "value" false).

  3. Build the outer leaf from the substituted tuple but with WHERE
     replaced by `true` — the substituted WHERE will be re-applied above
     the dep-join chain via a qpir-select wrapper so the sq references it
     contains are bound by the dep-joins.

  4. Chain a qpir-dep-join per marker. Left of the bottom dep-join is the
     outer leaf; each subsequent dep-join takes the previous as its left
     and the next marker's wrapped inner subquery as its right.

  5. If the original WHERE was non-trivial (not `true`), wrap the chain in
     a qpir-select carrying the substituted WHERE.

  6. If any field's expression was substituted (i.e., a field contained a
     scalar marker), wrap with qpir-map carrying the substituted fields.
     Otherwise the outer leaf already projects the right fields and the
     map wrapper is unnecessary.

  HAVING-side markers, inner_select_in, inner_select_exists, UNION ALL,
  and recursive subqueries are not yet handled — those shapes panic with
  a descriptive message per FAQ §1 (no silent fallback paths). Phase 3+
  add them.

Per FAQ §15: unhandled shapes must error loudly so the next phase
implementer knows exactly what to add.
*/

/* ==================== Marker detection ==================== */

/* qpl-marker-kind — return the inner_select kind symbol of expr, or nil. */
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

/* qpl-collect-markers — walk an expression tree and return every marker
subexpression, depth-first left-to-right. */
(define qpl-collect-markers (lambda (expr)
	(if (qpl-marker? expr)
		(list expr)
		(match expr
			(cons _sym args) (reduce (coalesceNil args '()) (lambda (acc a)
				(merge acc (qpl-collect-markers a))) '())
			'()))))

(define qpl-marker-subquery (lambda (marker)
	(match (qpl-marker-kind marker)
		(quote inner_select)         (nth marker 1)
		(quote inner_select_in)      (nth marker 2)
		(quote inner_select_exists)  (nth marker 1)
		nil)))

(define qpl-marker-lhs (lambda (marker)
	(match (qpl-marker-kind marker)
		(quote inner_select_in) (nth marker 1)
		nil)))

(define qpl-tuple-has-markers? (lambda (t) (begin
	(define collected (newsession))
	(collected "n" 0)
	(qpp-apply-to-tuple t (lambda (e) (begin
		(collected "n" (+ (collected "n") (count (qpl-collect-markers e))))
		e)))
	(> (collected "n") 0))))

/* ==================== IN / EXISTS rewrite (FAQ §11) ==================== */

/* Per FAQ §11: EXISTS/IN compile via COALESCE((SELECT COUNT(*) FROM …), 0) > 0.
The rewrite turns the non-scalar markers into a synthesized scalar inner_select
wrapping a COUNT subquery, so the substitution walker (which already handles
scalar markers) processes them uniformly.

   EXISTS (sub) →
     (> (coalesce (inner_select count-sub) 0) 0)
       where count-sub keeps sub's schema, tables, WHERE, GROUP BY but
       projects a single COUNT(*) named "value".

   a IN (sub) →
     (> (coalesce (inner_select count-sub) 0) 0)
       where count-sub keeps sub's schema, tables, GROUP BY but adds
       (equal?? a sub-first-field-expr) to WHERE and projects COUNT(*).

NULL semantics: tri-valued IN/NOT IN (FAQ §22, §24) is a phase 4 concern
when we add the match_count + null_count parallel COUNTs. For now this is
the strict (two-valued) rewrite. */

(define qpl-count-star-aggregate '((quote aggregate) 1 (quote +) 0))

(define qpl-and-cond (lambda (a b)
	(if (or (nil? a) (equal? a true)) b
		(if (or (nil? b) (equal? b true)) a
			(list (quote and) a b)))))

(define qpl-make-count-subquery-for-exists (lambda (sub)
	(qpp-rebuild-tuple
		(qpp-tuple-schema sub)
		(qpp-tuple-tables sub)
		(list (list "value" qpl-count-star-aggregate))
		(qpp-tuple-condition sub)
		(qpp-tuple-group sub)
		nil   /* HAVING dropped: the count above the group reduces to 0/n */
		'()   /* ORDER BY irrelevant for a scalar count */
		nil   /* LIMIT dropped */
		nil)))

(define qpl-make-count-subquery-for-in (lambda (a sub)
	(begin
		(define sub-fields (qpp-tuple-fields sub))
		(if (not (equal? (count sub-fields) 1))
			(error (concat "lift_dep_joins_pass: IN-subquery has " (string (count sub-fields))
				" projected fields; expected exactly 1. Multi-row IN is FAQ §22 territory and not yet implemented."))
			(begin
				(define sub-expr (nth (nth sub-fields 0) 1))
				(qpp-rebuild-tuple
					(qpp-tuple-schema sub)
					(qpp-tuple-tables sub)
					(list (list "value" qpl-count-star-aggregate))
					(qpl-and-cond
						(list (quote equal??) a sub-expr)
						(qpp-tuple-condition sub))
					(qpp-tuple-group sub)
					nil   /* HAVING dropped — see EXISTS comment */
					'() nil nil))))))

/* qpl-wrap-as-count-gt-zero — wrap a synthesized scalar inner_select in the
COALESCE-COUNT > 0 boolean shape per FAQ §11. */
(define qpl-wrap-as-count-gt-zero (lambda (count-sub)
	(list (quote >)
		(list (quote coalesce) (list (quote inner_select) count-sub) 0)
		0)))

/* qpl-rewrite-in-exists — walk an expression tree, rewrite every
inner_select_in / inner_select_exists into the COALESCE-COUNT > 0 form.
Leaves scalar inner_select untouched (it's already in the form the
substitution walker expects). */
(define qpl-rewrite-in-exists (lambda (expr) (begin
	(define k (qpl-marker-kind expr))
	(if (equal? k (quote inner_select_in))
		(qpl-wrap-as-count-gt-zero
			(qpl-make-count-subquery-for-in
				(qpl-rewrite-in-exists (qpl-marker-lhs expr))
				(qpl-marker-subquery expr)))
		(if (equal? k (quote inner_select_exists))
			(qpl-wrap-as-count-gt-zero
				(qpl-make-count-subquery-for-exists (qpl-marker-subquery expr)))
			(match expr
				(cons sym args) (cons sym (map (coalesceNil args '()) qpl-rewrite-in-exists))
				expr))))))

/* qpl-rewrite-in-exists-fields — apply the rewrite to each projection. */
(define qpl-rewrite-in-exists-fields (lambda (fields)
	(map (coalesceNil fields '()) (lambda (pair) (match pair
		'(name expr) (list name (qpl-rewrite-in-exists expr))
		pair)))))

(define qpl-rewrite-in-exists-group (lambda (group)
	(map (coalesceNil group '()) qpl-rewrite-in-exists)))

(define qpl-rewrite-in-exists-order (lambda (order)
	(map (coalesceNil order '()) (lambda (item) (match item
		'(expr dir) (list (qpl-rewrite-in-exists expr) dir)
		item)))))

/* qpl-rewrite-in-exists-tuple — apply qpl-rewrite-in-exists to every
expression slot of a 7-tuple. */
(define qpl-rewrite-in-exists-tuple (lambda (t) (qpp-rebuild-tuple
	(qpp-tuple-schema t)
	(qpp-tuple-tables t)
	(qpl-rewrite-in-exists-fields (qpp-tuple-fields t))
	(qpl-rewrite-in-exists (qpp-tuple-condition t))
	(qpl-rewrite-in-exists-group (qpp-tuple-group t))
	(qpl-rewrite-in-exists (qpp-tuple-having t))
	(qpl-rewrite-in-exists-order (qpp-tuple-order t))
	(qpp-tuple-limit t)
	(qpp-tuple-offset t))))

/* ==================== Substitution + collection ==================== */

(define qpl-sq-counter (newsession))
(qpl-sq-counter "n" 0)
(define qpl-fresh-sq-alias (lambda () (begin
	(qpl-sq-counter "n" (+ (qpl-sq-counter "n") 1))
	(concat "sq_" (string (qpl-sq-counter "n"))))))

/* qpl-sq-col — build the (get_column sq_alias false "value" false) reference
that replaces a scalar marker after lifting. */
(define qpl-sq-col (lambda (sq-alias)
	(list (quote get_column) sq-alias false "value" false)))

/* qpl-substitute-markers — walks an expression. Each scalar inner_select
encountered is replaced by a get_column reference; the marker's subquery is
recorded into `acc` (a newsession with key "list" → list of (sq-alias subquery)).
Non-scalar markers (IN/EXISTS) trigger an error — those are Phase 3+. */
(define qpl-substitute-markers (lambda (expr acc) (begin
	(define k (qpl-marker-kind expr))
	(if (equal? k (quote inner_select))
		(begin
			(define sq-alias (qpl-fresh-sq-alias))
			(acc "list" (merge (coalesceNil (acc "list") '())
				(list (list sq-alias (qpl-marker-subquery expr)))))
			(qpl-sq-col sq-alias))
		(if (not (nil? k))
			(error (concat "lift_dep_joins_pass: marker kind " (string k)
				" not yet supported (Phase 3+). Only scalar inner_select is handled."))
			(match expr
				(cons sym args) (cons sym (map (coalesceNil args '())
					(lambda (a) (qpl-substitute-markers a acc))))
				expr))))))

/* qpl-substitute-fields — apply qpl-substitute-markers to every projection
expression in a fields list, accumulating subqueries into acc. */
(define qpl-substitute-fields (lambda (fields acc)
	(map (coalesceNil fields '()) (lambda (pair) (match pair
		'(name expr) (list name (qpl-substitute-markers expr acc))
		pair)))))

/* qpl-substitute-group — apply to every group-by expression. */
(define qpl-substitute-group (lambda (group acc)
	(map (coalesceNil group '()) (lambda (e) (qpl-substitute-markers e acc)))))

/* qpl-substitute-order — apply to every order-by expression (preserving dir). */
(define qpl-substitute-order (lambda (order acc)
	(map (coalesceNil order '()) (lambda (item) (match item
		'(expr dir) (list (qpl-substitute-markers expr acc) dir)
		item)))))

/* qpl-fields-touched? — true if any projection in `orig` differs from `sub`
(meaning at least one field had a marker substituted). */
(define qpl-fields-touched? (lambda (orig sub)
	(not (equal? orig sub))))

/* ==================== Inner subquery wrapping ==================== */

/* qpl-wrap-inner-subquery-as-leaf — convert a parser-emitted inner subquery
7-tuple into a (qpir-leaf …) with its single output field renamed to "value"
so callers can reference it as (get_column sq_N false "value" false).

Currently requires a single-field inner subquery. Multi-field shapes (used by
IN/EXISTS in Phase 3+) get a different wrapping. */
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
				(error (concat "qpl-wrap-inner-subquery-as-leaf: inner subquery has "
					(string (count fields)) " fields; Phase 2 expects 1. "
					"Multi-field inner subqueries are Phase 3+ (IN/EXISTS).")))))))

/* ==================== Lift driver ==================== */

/* lift_dep_joins_pass — the L2 → L1 transformation.
Step 1 — pre-rewrite IN/EXISTS markers into the FAQ §11 COALESCE-COUNT > 0
shape; this turns them into scalar inner_selects so step 2 (substitution)
handles them uniformly. Step 2 — qpir-tree assembly via qpl-lift-with-markers. */
(define lift_dep_joins_pass (lambda (t)
	(if (not (qpp-tuple? t))
		(error "lift_dep_joins_pass: input is not a 7-tuple")
		(begin
			(define t-prime (qpl-rewrite-in-exists-tuple t))
			(if (not (qpl-tuple-has-markers? t-prime))
				(qpir-leaf t-prime)
				(qpl-lift-with-markers t-prime))))))

(define qpl-lift-with-markers (lambda (t) (begin
	/* Reject shapes Phase 2 does not yet handle: HAVING markers, group-by markers,
	   order-by markers. These need their own structural lifting. */
	(if (> (count (qpl-collect-markers (qpp-tuple-having t))) 0)
		(error "lift_dep_joins_pass: HAVING-level marker not yet supported (Phase 3+)") nil)
	(if (> (reduce (coalesceNil (qpp-tuple-group t) '()) (lambda (acc e)
			(+ acc (count (qpl-collect-markers e)))) 0) 0)
		(error "lift_dep_joins_pass: GROUP-BY-level marker not yet supported (Phase 3+)") nil)
	(if (> (reduce (coalesceNil (qpp-tuple-order t) '()) (lambda (acc item) (match item
			'(expr _dir) (+ acc (count (qpl-collect-markers expr)))
			acc)) 0) 0)
		(error "lift_dep_joins_pass: ORDER-BY-level marker not yet supported (Phase 3+)") nil)

	/* Collect + substitute all scalar markers from fields and condition.
	   Order matters: keep fields-first then condition-second so sq_N numbering
	   is deterministic for snapshot tests. */
	(define acc (newsession))
	(acc "list" '())
	(define orig-fields (qpp-tuple-fields t))
	(define sub-fields (qpl-substitute-fields orig-fields acc))
	(define orig-cond (qpp-tuple-condition t))
	(define sub-cond (qpl-substitute-markers orig-cond acc))
	(define markers (acc "list"))

	(if (equal? (count markers) 0)
		/* Nothing actually got lifted (defensive: should not happen because
		   qpl-tuple-has-markers? said yes). Fall back to leaf wrap. */
		(qpir-leaf t)
		(begin
			/* Build outer leaf: substituted fields, WHERE replaced by true
			   (real condition is re-applied above the dep-join chain so its
			   sq.value references can resolve to the dep-join's right side). */
			(define outer-leaf (qpir-leaf (qpp-rebuild-tuple
				(qpp-tuple-schema t)
				(qpp-tuple-tables t)
				sub-fields
				true
				(qpp-tuple-group t)
				(qpp-tuple-having t)
				(qpp-tuple-order t)
				(qpp-tuple-limit t)
				(qpp-tuple-offset t))))

			/* Chain qpir-dep-joins, one per marker. Bottom dep-join has outer-leaf
			   on the left; each subsequent dep-join's left is the previous chain
			   so each sq alias becomes visible above its point of introduction. */
			(define chained (reduce markers (lambda (left-acc pair) (match pair
				'(_sq-alias sub)
				(qpir-dep-join true left-acc (qpl-wrap-inner-subquery-as-leaf sub) '())
				left-acc))
				outer-leaf))

			/* Wrap with qpir-select if the original WHERE was non-trivial.
			   Comparing to `true` covers both literal-true and the parser's
			   default condition shape. */
			(define after-where
				(if (or (equal? sub-cond true) (equal? sub-cond (quote true)))
					chained
					(qpir-select sub-cond chained)))

			after-where)))))
