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
			(cons head args) (reduce (coalesceNil args '()) (lambda (acc a)
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
	(if (not (qpp-tuple? sub))
		(error "qpl-make-count-subquery-for-exists: sub is not a 7-tuple (likely UNION ALL — phase 5+)")
		(qpp-rebuild-tuple
			(qpp-tuple-schema sub)
			(qpp-tuple-tables sub)
			(list (list "value" qpl-count-star-aggregate))
			(qpp-tuple-condition sub)
			(qpp-tuple-group sub)
			nil   /* HAVING dropped: the count above the group reduces to 0/n */
			'()   /* ORDER BY irrelevant for a scalar count */
			nil   /* LIMIT dropped */
			nil))))

(define qpl-make-count-subquery-for-in (lambda (a sub)
	(if (not (qpp-tuple? sub))
		(error "qpl-make-count-subquery-for-in: sub is not a 7-tuple (likely UNION ALL — phase 5+)")
		(begin
			/* Normalize to pairs — sub may be parser-flat or pipeline-pairs. */
			(define sub-fields (qpp-fields-to-pairs (qpp-tuple-fields sub)))
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
					'() nil nil)))))))

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
	(if (nil? group) nil
		(map group qpl-rewrite-in-exists))))

(define qpl-rewrite-in-exists-order (lambda (order)
	(if (nil? order) nil
		(map order (lambda (item) (match item
			'(expr dir) (list (qpl-rewrite-in-exists expr) dir)
			item))))))

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

/* qpl-substitute-group — apply to every group-by expression.
Preserves nil as nil — legacy distinguishes nil-group ("no GROUP BY",
top-level scalar) from empty-list ("explicit empty key set"). */
(define qpl-substitute-group (lambda (group acc)
	(if (nil? group) nil
		(map group (lambda (e) (qpl-substitute-markers e acc))))))

/* qpl-substitute-order — apply to every order-by expression (preserving dir).
Preserves nil as nil. */
(define qpl-substitute-order (lambda (order acc)
	(if (nil? order) nil
		(map order (lambda (item) (match item
			'(expr dir) (list (qpl-substitute-markers expr acc) dir)
			item))))))

/* qpl-fields-touched? — true if any projection in `orig` differs from `sub`
(meaning at least one field had a marker substituted). */
(define qpl-fields-touched? (lambda (orig sub)
	(not (equal? orig sub))))

/* ==================== Inner subquery decomposition ==================== */

/* qpl-is-aggregate-expr? — true when expr is a bare `(aggregate inner reducer init)`
form as emitted by the parser for SUM/COUNT (and the wrapped inner of AVG). */
(define qpl-is-aggregate-expr? (lambda (expr) (match expr
	(cons head rest) (match head
		(symbol aggregate)         true
		(quote aggregate)          true
		'(quote aggregate)         true
		'aggregate                 true
		false)
	false)))

/* qpl-expr-has-aggregate? — true if expr is or contains an aggregate
subexpression anywhere. Used to detect "complex aggregate expressions"
(e.g. AVG = SUM/COUNT, or SUM(x)+1) which Phase 4 does not yet decompose. */
(define qpl-expr-has-aggregate? (lambda (expr)
	(if (qpl-is-aggregate-expr? expr)
		true
		(match expr
			(cons head args) (reduce (coalesceNil args '()) (lambda (acc a)
				(or acc (qpl-expr-has-aggregate? a))) false)
			false))))

/* qpl-collect-aggregates-in-fields — return the list of aggregate fields
(those whose expression is a bare aggregate). Each entry: (field-name agg-expr). */
(define qpl-collect-aggregates-in-fields (lambda (fields)
	(reduce (coalesceNil fields '()) (lambda (acc pair) (match pair
		'(name expr) (if (qpl-is-aggregate-expr? expr)
			(merge acc (list (list name expr)))
			acc)
		acc)) '())))

/* qpl-collect-non-aggregate-fields — return the list of non-aggregate fields
in the same order they appear. Errors loudly if a field contains an aggregate
nested inside a non-bare expression (e.g. AVG = (SUM/COUNT)) — phase 5 will
handle those. */
(define qpl-collect-non-aggregate-fields (lambda (fields)
	(reduce (coalesceNil fields '()) (lambda (acc pair) (match pair
		'(name expr) (begin
			(if (qpl-is-aggregate-expr? expr)
				acc
				(begin
					(if (qpl-expr-has-aggregate? expr)
						(error (concat "lift_dep_joins_pass: field '" name
							"' contains a nested aggregate inside a compound expression. "
							"Phase 4 only decomposes BARE aggregate fields; mixed "
							"shapes like AVG, SUM(x)+1 are Phase 5+."))
						nil)
					(merge acc (list pair)))))
		acc)) '())))

/* qpl-leaf-input-fields-for-aggs — produce the projection list the underlying
qpir-leaf must expose so the qpir-groupby above can compute its aggregates.

For an aggregate (aggregate inner reducer init):
  - The leaf must project the `inner` expression so the agg reads it.
  - We synthesize a name `agg-in-N` per aggregate and the qpir-groupby's aggs
    list references that name via get_column.

For Phase 4 we use a simpler convention: each aggregate's `inner` expression
is projected under its FIELD NAME — so for `(total SUM(amount))` the leaf
projects `(total amount)` and the qpir-groupby's agg is `(total (aggregate
(get_column leaf-alias false "total" false) + 0))`.

That keeps the lowering trivial: the leaf is a scan that exposes columns
named by their visible alias, the groupby reduces them into the same names.

Returns: (list-of-leaf-fields  list-of-rewritten-groupby-aggs). */
(define qpl-leaf-and-agg-projections (lambda (agg-fields)
	(reduce agg-fields (lambda (acc pair) (match pair
		'(name (cons head rest))
		(begin
			(define agg-args rest)
			(define agg-inner (nth agg-args 0))
			(define agg-reducer (nth agg-args 1))
			(define agg-init (nth agg-args 2))
			/* The leaf projects the aggregate's inner expression under `name`. */
			(define leaf-field (list name agg-inner))
			/* The groupby reads back `name` from its child via a get_column ref;
			   the synthesised alias "" stays empty — the lowering will resolve
			   it against the leaf's projected columns. */
			(define agg-after (list name
				(list (quote aggregate)
					(list (quote get_column) "" false name false)
					agg-reducer agg-init)))
			(list
				(merge (nth acc 0) (list leaf-field))
				(merge (nth acc 1) (list agg-after))))
		acc))
		(list '() '()))))

/* qpl-needs-decompose? — true if the subquery has aggregates in its fields
or a non-empty GROUP BY. Phase 4 also rejects HAVING/ORDER/LIMIT inner
subqueries because they need additional wrappers (phase 5+).
NOTE: (nil? '()) is FALSE in this dialect — use (> (count ...) 0) instead. */
(define qpl-needs-decompose? (lambda (sub)
	(or
		(> (count (qpl-collect-aggregates-in-fields (qpp-tuple-fields sub))) 0)
		(> (count (coalesceNil (qpp-tuple-group sub) '())) 0))))

/* qpl-wrap-inner-subquery — convert a parser-emitted inner subquery 7-tuple
into a Layer-1 IR subtree exposing one column named "value".

Cases:
  (1) No aggregates, no GROUP BY → single qpir-leaf with the field renamed
      to "value". (Already what phase 1-3 did.)
  (2) Single bare-aggregate field, no GROUP BY (static-group case) →
      (qpir-groupby '() ((value <agg>)) nil (qpir-leaf {…}))
      The leaf projects the aggregate's inner expression as "value", the
      groupby aggregates it.
  (3) Multiple fields, GROUP BY, HAVING, complex agg expressions → not yet
      supported, errors loudly per FAQ §1. */
/* qpl-rename-first-field-to-value — produce a copy of sub where the single
visible field is named "value" so callers can reference the scalar subquery's
output as `(get_column sq_N false "value" false)` regardless of the user's
original SQL alias. */
(define qpl-rename-first-field-to-value (lambda (sub) (begin
	/* Sub's fields can be EITHER flat (name1 expr1 …) (parser shape) or
	   list-of-pairs ((name1 expr1) …) (pipeline-internal shape). Normalize
	   to pairs via qpp-fields-to-pairs before counting/extracting. */
	(define fields-as-pairs (qpp-fields-to-pairs (qpp-tuple-fields sub)))
	(if (not (equal? (count fields-as-pairs) 1))
		(error (concat "qpl-rename-first-field-to-value: expected 1 field, found "
			(string (count fields-as-pairs)))) nil)
	(define field-pair (nth fields-as-pairs 0))
	(define field-expr (nth field-pair 1))
	(qpp-rebuild-tuple
		(qpp-tuple-schema sub)
		(qpp-tuple-tables sub)
		(list (list "value" field-expr))
		(qpp-tuple-condition sub)
		(qpp-tuple-group sub)
		(qpp-tuple-having sub)
		(qpp-tuple-order sub)
		(qpp-tuple-limit sub)
		(qpp-tuple-offset sub)))))

(define qpl-wrap-inner-subquery (lambda (sub)
	(if (not (qpp-tuple? sub))
		(error "qpl-wrap-inner-subquery: sub is not a 7-tuple")
		(begin
			/* Normalize to pairs — sub may be parser-flat or pipeline-pairs. */
			(define fields (qpp-fields-to-pairs (qpp-tuple-fields sub)))
			(if (not (equal? (count fields) 1))
				(error (concat "qpl-wrap-inner-subquery: inner subquery has "
					(string (count fields)) " fields; expected exactly 1. "
					"Multi-field inner subqueries are phase 5+.")) nil)
			(if (not (nil? (qpp-tuple-having sub)))
				(error "qpl-wrap-inner-subquery: inner subquery with HAVING not yet supported (phase 5+)") nil)
			/* Step 1: rename the visible field to "value" so callers uniformly
			   reference sq_N.value regardless of the user's SQL alias. */
			(define renamed (qpl-rename-first-field-to-value sub))
			/* Step 2: RECURSIVELY lift the renamed sub. This is the architectural
			   fix per FAQ "every query is unnestable": if `sub` itself contains
			   inner_select markers (a NESTED correlated subquery), lift turns
			   them into qpir-dep-join nodes in the right subtree. The outer
			   dep-join (built by the caller) then wraps this whole tree, and
			   unnest_pass eliminates BOTH the outer dep-join and the inner
			   ones (top-down per BTW2025 §3.2 with parent-chained UnnestingInfo
			   — see queryplan-unnest.scm). */
			(define lifted (lift_dep_joins_pass renamed))
			/* Step 3: if lifted is a plain qpir-leaf whose 7-tuple needs
			   aggregate/group-by decomposition (the static-group case for the
			   typical SUM correlated subquery), apply the decomposition so the
			   §3.3 groupby rule has a target during unnest. If lifted is a
			   richer tree (because the sub had its own markers), return as-is —
			   any aggregates are already operator-level inside that tree. */
			(if (equal? (qpir-kind lifted) (quote qpir-leaf))
				(begin
					(define leaf-tuple (qpir-leaf-7tuple lifted))
					(if (qpl-needs-decompose? leaf-tuple)
						(qpl-build-groupby-wrapped-inner leaf-tuple)
						(qpl-build-simple-leaf-inner leaf-tuple)))
				lifted)))))

/* qpl-condition-is-trivial? — true when a WHERE condition is `true` (or
the literal true symbol); such conditions don't need to be hoisted. */
(define qpl-condition-is-trivial? (lambda (cond)
	(or (nil? cond) (equal? cond true) (equal? cond (quote true)))))

/* qpl-wrap-with-select-if-needed — if cond is non-trivial, return
(qpir-select cond inner); else return inner unchanged. Used to hoist the
inner-subquery's WHERE into operator-level so the BTW2025 §3.3 select rule
can apply during unnest_pass. */
(define qpl-wrap-with-select-if-needed (lambda (cond inner)
	(if (qpl-condition-is-trivial? cond)
		inner
		(qpir-select cond inner))))

(define qpl-build-simple-leaf-inner (lambda (sub) (begin
	(define field-pair (nth (qpp-tuple-fields sub) 0))
	(define field-expr (nth field-pair 1))
	(if (qpl-expr-has-aggregate? field-expr)
		(error "qpl-build-simple-leaf-inner: field has nested aggregate; should have decomposed")
		/* Hoist WHERE to qpir-select wrapper (architectural — gives the
		   unnest §3.3 select rule a place to fire). The leaf below keeps
		   only the table scan + projection. */
		(qpl-wrap-with-select-if-needed
			(qpp-tuple-condition sub)
			(qpir-leaf (qpp-rebuild-tuple
				(qpp-tuple-schema sub)
				(qpp-tuple-tables sub)
				(list (list "value" field-expr))
				true
				(qpp-tuple-group sub)
				nil
				(qpp-tuple-order sub)
				(qpp-tuple-limit sub)
				(qpp-tuple-offset sub))))))))

(define qpl-build-groupby-wrapped-inner (lambda (sub) (begin
	(define agg-fields (qpl-collect-aggregates-in-fields (qpp-tuple-fields sub)))
	(define non-agg-fields (qpl-collect-non-aggregate-fields (qpp-tuple-fields sub)))
	/* Phase 4 only supports the single-bare-aggregate case (the most common
	   shape for scalar correlated subqueries and the FAQ §11 COUNT rewrite). */
	(if (not (equal? (count agg-fields) 1))
		(error (concat "qpl-build-groupby-wrapped-inner: expected 1 aggregate field, found "
			(string (count agg-fields)) " — multiple aggregates per inner subquery is phase 5+")) nil)
	(if (> (count non-agg-fields) 0)
		(error "qpl-build-groupby-wrapped-inner: mixed agg + non-agg fields not supported in phase 4") nil)
	(define agg-pair (nth agg-fields 0))
	(define agg-name (nth agg-pair 0))
	(define agg-expr (nth agg-pair 1))
	(define agg-args (cdr agg-expr))
	(define agg-inner (nth agg-args 0))
	(define agg-reducer (nth agg-args 1))
	(define agg-init (nth agg-args 2))
	/* Per FAQ "canonical names / helper identities derived from physical source
	   columns": don't synthesize placeholder aliases. The aggregate keeps its
	   original column refs (e.g. (get_column pi amount)); the leaf below must
	   project every column referenced by the aggregate's inner expression AND
	   the WHERE predicate so the runtime can read them at scan time.

	   The qpir-groupby outputs its aggregate under the name "value" so callers
	   uniformly reference sq.value. The WHERE clause is hoisted to a qpir-select
	   between groupby and leaf so the unnest §3.3 select rule can fire. */
	(define leaf-cols-from-agg (qpir-expr-column-refs agg-inner))
	(define leaf-cols-from-where
		(qpir-expr-column-refs (coalesceNil (qpp-tuple-condition sub) true)))
	(define leaf-cols (qpl-dedupe-col-refs (merge leaf-cols-from-agg leaf-cols-from-where)))
	/* Materialize the leaf's projections. Only project columns that the leaf
	   can actually provide — i.e. those whose tblvar matches one of sub's tables.
	   Outer-ref columns stay as free vars and are resolved by the dep-join. */
	(define leaf-aliases (map (coalesceNil (qpp-tuple-tables sub) '()) (lambda (td)
		(if (or (nil? td) (< (count td) 1)) nil (nth td 0)))))
	(define leaf-fields (map
		(filter leaf-cols (lambda (ref) (match ref
			'(tv col) (has? leaf-aliases tv)
			false)))
		(lambda (ref) (match ref
			'(tv col) (list col (list (quote get_column) tv false col false))
			ref))))
	(define leaf-bare (qpir-leaf (qpp-rebuild-tuple
		(qpp-tuple-schema sub)
		(qpp-tuple-tables sub)
		leaf-fields
		true
		'()  /* GROUP BY moves up to qpir-groupby */
		nil
		'()  /* ORDER BY irrelevant for an aggregate scalar */
		nil  /* LIMIT same */
		nil)))
	(define leaf-with-select (qpl-wrap-with-select-if-needed
		(qpp-tuple-condition sub)
		leaf-bare))
	(define group-keys (coalesceNil (qpp-tuple-group sub) '()))
	(qpir-groupby
		group-keys
		(list (list "value"
			(list (quote aggregate) agg-inner agg-reducer agg-init)))
		nil
		leaf-with-select))))

/* qpl-dedupe-col-refs — remove duplicate (tv col) pairs from a list. */
(define qpl-dedupe-col-refs (lambda (refs)
	(reduce refs (lambda (acc ref)
		(if (has? acc ref) acc (merge acc (list ref))))
		'())))

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
			'(expr dir) (+ acc (count (qpl-collect-markers expr)))
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
				'(sq-alias sub)
				(qpir-dep-join true left-acc (qpl-wrap-inner-subquery sub) '() sq-alias)
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
