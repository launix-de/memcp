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

/* qpl-and-conjuncts — flatten a nested (and a b c …) tree into a list of leaf
conjuncts. Trivial cases (nil / true) become an empty list. Used by the
correlated-LIMIT detector to scan WHERE for equality conjuncts that bind
outer-refs to inner columns. */
(define qpl-and-conjuncts (lambda (expr)
	(if (or (nil? expr) (equal? expr true)) '()
		(match expr
			(cons head args) (begin
				(define is-and (match head
					(symbol and)  true
					(quote and)   true
					'(quote and)  true
					'and          true
					false))
				(if is-and
					(reduce (coalesceNil args '()) (lambda (acc a)
						(merge acc (qpl-and-conjuncts a))) '())
					(list expr)))
			(list expr)))))

/* qpl-ref-bound-by-equality? — true if `ref-pair` (tv col) appears on one
side of any equality conjunct in `conjuncts` where the OTHER side is a
column ref to a non-outer alias (= inner-bound). This is FAQ §38's "simple
unnesting" condition: when an outer-ref is equi-bound to an inner column,
the correlation is provided by the join condition itself, and a LIMIT on
the sub becomes per-outer-binding implicitly. */
(define qpl-ref-bound-by-equality? (lambda (ref-pair conjuncts inner-aliases) (begin
	(define is-eq-sym (lambda (sym) (match sym
		(symbol equal??)  true
		(quote equal??)   true
		'(quote equal??)  true
		'equal??          true
		(symbol equal?)   true
		(quote equal?)    true
		'(quote equal?)   true
		'equal?           true
		(symbol =)        true
		(quote =)         true
		'(quote =)        true
		'=                true
		false)))
	(define ref-key (lambda (e) (match e
		'((symbol get_column) tv ti col ci) (list tv col)
		'((quote get_column)  tv ti col ci) (list tv col)
		nil)))
	(reduce conjuncts (lambda (acc c) (or acc (match c
		(cons head args) (if (and (is-eq-sym head) (equal? (count args) 2))
			(begin
				(define a (nth args 0))
				(define b (nth args 1))
				(define ka (ref-key a))
				(define kb (ref-key b))
				(if (or (nil? ka) (nil? kb)) false
					/* Check both orderings: ref on left/inner on right, or vice versa. */
					(or (and (equal? ka ref-pair) (has? inner-aliases (nth kb 0)))
					    (and (equal? kb ref-pair) (has? inner-aliases (nth ka 0))))))
			false)
		false))) false))))

/* qpl-extract-col-refs — like qpir-expr-column-refs but doesn't skip nil-tv.
Returns a list of (tv col) pairs for every (get_column …) leaf. */
(define qpl-extract-col-refs (lambda (expr)
	(match expr
		'((symbol get_column) tv ti col ci) (list (list tv col))
		'((quote get_column)  tv ti col ci) (list (list tv col))
		(cons head args) (reduce (coalesceNil args '()) (lambda (acc a)
			(merge acc (qpl-extract-col-refs a))) '())
		'())))

/* qpl-drop-redundant-correlated-limit — for an inner_select's sub-tuple,
if it has LIMIT k AND its WHERE has equality conjuncts that fully equi-bind
every outer-ref to an inner column (FAQ §38 "simple unnesting" condition,
trivial dep-join after equi-binding per FAQ §39), the LIMIT applies
per-outer-binding implicitly via the equality and can be safely dropped.

Without this, my pipeline lowers LIMIT k GLOBALLY on the derived sub,
which limits before the outer-correlation join — yielding wrong rows.

Returns the sub-tuple with LIMIT dropped if conditions are met, else
returns sub unchanged. This is NOT a substitute for the general FAQ §43
ROW_NUMBER PARTITION rewrite — when outer-refs are NOT equi-bound (e.g.
INEQUALITY correlation, or no WHERE-equality binding) the LIMIT genuinely
needs per-outer-binding window semantics that this drop does not provide. */
(define qpl-drop-redundant-correlated-limit (lambda (sub)
	(if (not (qpp-tuple? sub)) sub
		(begin
			(define lim (qpp-tuple-limit sub))
			(define off (qpp-tuple-offset sub))
			(define ord (qpp-tuple-order sub))
			/* Drop ONLY when: LIMIT is set, no OFFSET, no ORDER BY.
			   ORDER BY means the LIMIT selects a SPECIFIC subset (e.g. top-k);
			   dropping the LIMIT would change which rows are returned, even
			   if equi-binding bounds the cardinality. The ROW_NUMBER PARTITION
			   rewrite (qpl-rewrite-correlated-limit-with-rownumber) handles
			   the ordered case correctly. */
			(if (or (nil? lim) (not (nil? off))
				(and (not (nil? ord)) (> (count ord) 0))) sub
				(begin
					(define inner-aliases (qpl-outer-aliases (qpp-tuple-tables sub)))
					(define cond (qpp-tuple-condition sub))
					(define conjuncts (qpl-and-conjuncts cond))
					/* All column refs in the WHERE that are NOT bound by an inner
					   alias are outer-refs. We require EVERY such outer-ref to be
					   equi-bound to an inner column. */
					(define all-refs (qpl-extract-col-refs cond))
					(define outer-refs (filter all-refs (lambda (rp) (match rp
						'(tv col) (not (has? inner-aliases tv))
						false))))
					(if (equal? (count outer-refs) 0) sub
						(begin
							(define all-bound (reduce outer-refs (lambda (acc ref)
								(and acc (qpl-ref-bound-by-equality? ref conjuncts inner-aliases)))
								true))
							(if all-bound
								(qpp-rebuild-tuple
									(qpp-tuple-schema sub)
									(qpp-tuple-tables sub)
									(qpp-tuple-fields sub)
									cond
									(qpp-tuple-group sub)
									(qpp-tuple-having sub)
									(qpp-tuple-order sub)
									nil   /* LIMIT dropped — equi-binding makes it per-outer-binding */
									nil)
								sub))))))))))

/* qpl-rewrite-redundant-limit-in-expr — walk expr, find inner_select markers,
apply qpl-drop-redundant-correlated-limit to their sub-tuples. */
(define qpl-rewrite-redundant-limit-in-expr (lambda (expr)
	(match expr
		(cons sym args) (begin
			(define is-scalar (match sym
				(symbol inner_select) true
				(quote inner_select)  true
				'(quote inner_select) true
				'inner_select         true
				false))
			(if (and is-scalar (equal? (count args) 1))
				(list sym (qpl-drop-redundant-correlated-limit (nth args 0)))
				(cons sym (map (coalesceNil args '())
					(lambda (a) (qpl-rewrite-redundant-limit-in-expr a))))))
		expr)))

(define qpl-rewrite-redundant-limit-tuple (lambda (t)
	(qpp-apply-to-tuple t qpl-rewrite-redundant-limit-in-expr)))

/* ==================== FAQ §43 ROW_NUMBER PARTITION rewrite ==================== */

/* qpl-rewrite-correlated-limit-with-rownumber — for an inner_select's
sub-tuple that has LIMIT k [OFFSET o] AND outer-correlation that
qpl-drop-redundant-correlated-limit could NOT eliminate, rewrite the sub
per FAQ §43 using a ROW_NUMBER OVER (PARTITION BY <outer-refs>) wrapper.

Transformation:
  sub:  SELECT X FROM T WHERE corr [ORDER BY o] LIMIT k [OFFSET off]
        (with outer-refs in corr)
  →
  wrapper:
    SELECT __value
    FROM (
      SELECT X AS __value,
             ROW_NUMBER() OVER (PARTITION BY <outer-refs> ORDER BY o) AS __rn
      FROM T WHERE corr
    ) AS __limit_wrap
    WHERE __rn BETWEEN off+1 AND k+off

The partition-by carries the outer-refs (still correlated). After my
pipeline unnests the outer correlation, the partition keys align with the
outer-binding domain so the LIMIT applies PER OUTER ROW (FAQ §43 "must
hold per outer binding, not globally").

Conditions for rewrite:
  - sub has LIMIT k (non-nil)
  - sub is correlated (has outer-refs in WHERE, fields, group, etc.)
  - sub is NOT already handled by qpl-drop-redundant-correlated-limit
    (caller must check the drop didn't fire — i.e. sub.limit is still set)

Output: a new sub-tuple containing one fields entry (renamed __value),
condition rn-filter, no ORDER BY (moved into window) / no LIMIT (moved up).
The inner derived `__limit_wrap` carries the window function + original
WHERE.

This is the general case complement to qpl-drop-redundant-correlated-limit
which handles the equi-binding optimization case. */

/* qpl-uniq-counter / qpl-fresh-limwrap — generate unique sub-alias names. */
(define qpl-limwrap-counter (newsession))
(qpl-limwrap-counter "n" 0)
(define qpl-fresh-limwrap-alias (lambda () (begin
	(qpl-limwrap-counter "n" (+ (qpl-limwrap-counter "n") 1))
	(concat "__limit_wrap_" (string (qpl-limwrap-counter "n"))))))

/* qpl-extract-col-refs-skip-nested — like qpl-extract-col-refs but does NOT
descend into nested inner_select / inner_select_in / inner_select_exists
markers. Refs inside those sub-tuples are inner to the nested scope, not
free at this level. Without this, a nested sub's WHERE refs are wrongly
classified as "outer refs" of the outer sub. */
(define qpl-marker-head? (lambda (head) (match head
	(symbol inner_select)         true
	(quote inner_select)          true
	'(quote inner_select)         true
	'inner_select                 true
	(symbol inner_select_in)      true
	(quote inner_select_in)       true
	'(quote inner_select_in)      true
	'inner_select_in              true
	(symbol inner_select_exists)  true
	(quote inner_select_exists)   true
	'(quote inner_select_exists)  true
	'inner_select_exists          true
	false)))

(define qpl-extract-col-refs-skip-nested (lambda (expr)
	(match expr
		'((symbol get_column) tv ti col ci) (list (list tv col))
		'((quote get_column)  tv ti col ci) (list (list tv col))
		(cons head args)
			(if (qpl-marker-head? head)
				/* Stop descent at nested-sub markers — their refs are inner-scope. */
				'()
				(reduce (coalesceNil args '()) (lambda (acc a)
					(merge acc (qpl-extract-col-refs-skip-nested a))) '()))
		'())))

/* qpl-sub-outer-refs — collect (tv col) pairs in sub's expressions that are
NOT bound by any of sub's table aliases. Walks WHERE, fields, group, having,
order. Deduplicates by (tv col). Skips nested inner_select markers so refs
inside nested subs aren't wrongly classified as outer-refs at this level. */
(define qpl-sub-outer-refs (lambda (sub) (begin
	(define inner-aliases (qpl-outer-aliases (qpp-tuple-tables sub)))
	(define all-cond-refs (qpl-extract-col-refs-skip-nested
		(qpp-tuple-condition sub)))
	(define all-fields-refs (reduce
		(qpp-fields-to-pairs (qpp-tuple-fields sub))
		(lambda (acc pair) (match pair
			'(name expr) (merge acc (qpl-extract-col-refs-skip-nested expr))
			acc))
		'()))
	(define all-refs (merge all-cond-refs all-fields-refs))
	(define outer-refs (filter all-refs (lambda (rp) (match rp
		'(tv col) (and (not (nil? tv)) (not (has? inner-aliases tv)))
		false))))
	/* Deduplicate */
	(reduce outer-refs (lambda (acc rp)
		(if (has? acc rp) acc (merge acc (list rp))))
		'()))))

/* qpl-find-equibind-inner-col — for an outer-ref (tv, col), find an
equality conjunct in WHERE that binds it to an INNER column. Returns
(inner-tv, inner-col) or nil. Used by qpl-build-rownumber-window so
PARTITION BY uses the inner-equivalent column (FAQ §35 canonical names)
when available, avoiding correlated PARTITION BY that legacy's window
path can't handle inside derived sub-tuples. */
(define qpl-find-equibind-inner-col (lambda (where outer-ref inner-aliases)
	(reduce (qpl-and-conjuncts where) (lambda (found conj)
		(if (not (nil? found)) found
			(match conj
				'((symbol equal??) lhs rhs) (qpl-equibind-pair lhs rhs outer-ref inner-aliases)
				'((quote equal??)  lhs rhs) (qpl-equibind-pair lhs rhs outer-ref inner-aliases)
				'((symbol =)       lhs rhs) (qpl-equibind-pair lhs rhs outer-ref inner-aliases)
				'((quote =)        lhs rhs) (qpl-equibind-pair lhs rhs outer-ref inner-aliases)
				nil)))
		nil)))

(define qpl-equibind-pair (lambda (lhs rhs outer-ref inner-aliases)
	(begin
		(define li (qpl-col-ref-info lhs))
		(define ri (qpl-col-ref-info rhs))
		(define out-tv (nth outer-ref 0))
		(define out-col (nth outer-ref 1))
		(if (and (not (nil? li))
				 (equal? (nth li 0) out-tv) (equal? (nth li 1) out-col)
				 (not (nil? ri))
				 (has? inner-aliases (nth ri 0)))
			ri
			(if (and (not (nil? ri))
					 (equal? (nth ri 0) out-tv) (equal? (nth ri 1) out-col)
					 (not (nil? li))
					 (has? inner-aliases (nth li 0)))
				li
				nil)))))

(define qpl-col-ref-info (lambda (expr) (match expr
	'((symbol get_column) tv ti col ci) (list tv col)
	'((quote get_column)  tv ti col ci) (list tv col)
	nil)))

(define qpl-and-conjuncts (lambda (expr) (match expr
	(cons head args)
		(if (or (equal? head (quote and)) (equal? head 'and))
			(reduce args (lambda (acc a)
				(merge acc (qpl-and-conjuncts a))) '())
			(list expr))
	(if (nil? expr) '() (list expr)))))

/* qpl-build-rownumber-window — build the window_func node for ROW_NUMBER
with PARTITION BY <outer-refs> ORDER BY <order-items>.

Per FAQ §35 canonical names: when an outer-ref is equi-bound to an inner
column in the sub's WHERE, use the inner column in PARTITION BY. This
turns a correlated window (outer-ref in PARTITION BY) into an uncorrelated
window over the inner table alone — which legacy's window-function path
handles correctly inside derived sub-tuples. */
(define qpl-build-rownumber-window (lambda (outer-refs order-items where inner-aliases)
	(begin
		(define partition-exprs (map outer-refs (lambda (rp) (match rp
			'(tv col) (begin
				(define inner-equiv (qpl-find-equibind-inner-col where rp inner-aliases))
				(if (not (nil? inner-equiv))
					(list (quote get_column) (nth inner-equiv 0) false
						(nth inner-equiv 1) false)
					(list (quote get_column) tv false col false)))
			rp))))
		(define order-list (if (nil? order-items) '() order-items))
		(list (quote window_func) "ROW_NUMBER" '()
			(list partition-exprs order-list)))))

/* qpl-split-where-by-correlation — split a WHERE expression into
(correlated-conjuncts, uncorrelated-conjuncts). A conjunct is "correlated"
when it references any outer-ref (passed in). Used by the FAQ §43 rewrite
to keep the correlation filter at the WRAPPER level (outside the window-
computing inner sub) — legacy's window-function path handles uncorrelated
inner subs cleanly, but breaks on correlated-WHERE inside a window-bearing
derived. */
(define qpl-expr-refs-any-outer? (lambda (expr outer-ref-pairs)
	(reduce (qpl-extract-col-refs-skip-nested expr) (lambda (acc rp)
		(or acc (has? outer-ref-pairs rp))) false)))

(define qpl-split-where-by-correlation (lambda (where outer-ref-pairs)
	(begin
		(define conjs (qpl-and-conjuncts where))
		(define corr (filter conjs (lambda (c)
			(qpl-expr-refs-any-outer? c outer-ref-pairs))))
		(define uncorr (filter conjs (lambda (c)
			(not (qpl-expr-refs-any-outer? c outer-ref-pairs)))))
		(list
			(reduce corr qpl-and-cond nil)
			(reduce uncorr qpl-and-cond nil)))))

(define qpl-rewrite-correlated-limit-with-rownumber (lambda (sub)
	(if (not (qpp-tuple? sub)) sub
		(begin
			(define lim (qpp-tuple-limit sub))
			(define off (qpp-tuple-offset sub))
			(if (nil? lim) sub
				(begin
					(define outer-refs (qpl-sub-outer-refs sub))
					(if (equal? (count outer-refs) 0) sub
						(begin
							/* Build inner-with-window: same as sub but add __rn field.
							   Field format is FLAT (parser shape) so downstream
							   reduce_assoc consumers receive the expected even-length
							   dict. */
							(define sub-fields-pairs (qpp-fields-to-pairs (qpp-tuple-fields sub)))
							(if (not (equal? (count sub-fields-pairs) 1))
								(error (concat
									"qpl-rewrite-correlated-limit-with-rownumber: expected 1 field, "
									"found " (string (count sub-fields-pairs))
									" — multi-field LIMIT rewrite is phase 5+")) nil)
							(define orig-field-pair (nth sub-fields-pairs 0))
							(define orig-field-name (nth orig-field-pair 0))
							(define orig-field-expr (nth orig-field-pair 1))
							(define order-items (qpp-tuple-order sub))
							(define sub-inner-aliases (qpl-outer-aliases (qpp-tuple-tables sub)))
							(define window-expr (qpl-build-rownumber-window
								outer-refs order-items
								(qpp-tuple-condition sub) sub-inner-aliases))
							/* Split WHERE: keep uncorrelated inside the window sub,
							   move correlated OUT to the wrapper. Legacy's window-
							   function path handles uncorrelated derived cleanly.
							   The window's PARTITION BY (using inner-equibound col
							   per FAQ §35) already encodes the per-outer-group
							   partitioning, so moving the correlation filter to the
							   wrapper preserves semantics. Also project the inner
							   equi-bound cols as passthrough so the wrapper's
							   correlation conjunct can reference them. */
							(define split (qpl-split-where-by-correlation
								(qpp-tuple-condition sub) outer-refs))
							(define corr-where (nth split 0))
							(define uncorr-where (nth split 1))
							/* Collect inner-equibound passthroughs needed by corr-where. */
							(define passthrough-cols
								(qpl-collect-corr-passthroughs corr-where sub-inner-aliases))
							(define passthrough-pairs (reduce passthrough-cols
								(lambda (acc col-ref) (match col-ref
									'(tv col) (merge acc (list col
										(list (quote get_column) tv false col false)))
									acc)) '()))
							/* FLAT fields: __value, __rn, plus passthrough cols. */
							(define inner-sub-fields (merge
								(list "__value" orig-field-expr
									"__rn" window-expr)
								passthrough-pairs))
							(define inner-sub (qpp-rebuild-tuple
								(qpp-tuple-schema sub)
								(qpp-tuple-tables sub)
								inner-sub-fields
								uncorr-where   /* only uncorrelated WHERE inside */
								(qpp-tuple-group sub)
								(qpp-tuple-having sub)
								nil    /* ORDER moved into window */
								nil    /* LIMIT moved up */
								nil))  /* OFFSET moved up */
							(define wrap-alias (qpl-fresh-limwrap-alias))
							(define schema (qpp-tuple-schema sub))
							(define offset-val (if (nil? off) 0 off))
							/* Wrapper WHERE: rn-filter AND retargeted corr-where.
							   Retarget refs to inner cols → wrap-alias (since they're
							   now projected via the wrapper's __limit_wrap). */
							(define rn-ref (list (quote get_column) wrap-alias false "__rn" false))
							(define rn-condition
								(if (nil? off)
									(list (quote <=) rn-ref lim)
									(list (quote and)
										(list (quote >) rn-ref offset-val)
										(list (quote <=) rn-ref (list (quote +) lim offset-val)))))
							(define retargeted-corr
								(if (nil? corr-where) nil
									(qpl-retarget-refs corr-where sub-inner-aliases wrap-alias)))
							(define wrapper-where
								(if (nil? retargeted-corr) rn-condition
									(list (quote and) rn-condition retargeted-corr)))
							(qpp-rebuild-tuple
								schema
								(list (list wrap-alias schema inner-sub false nil))
								/* FLAT fields for wrapper too. */
								(list orig-field-name
									(list (quote get_column) wrap-alias false "__value" false))
								wrapper-where
								nil nil nil nil nil))))))))))

/* qpl-collect-corr-passthroughs — find inner-side col refs in a correlated
WHERE expression, return the list of (tv col) pairs that need to be projected
through the window-bearing derived so the wrapper can reference them. */
(define qpl-collect-corr-passthroughs (lambda (expr inner-aliases)
	(if (nil? expr) '()
		(begin
			(define all-refs (qpl-extract-col-refs-skip-nested expr))
			(define inner-refs (filter all-refs (lambda (rp) (match rp
				'(tv col) (has? inner-aliases tv)
				false))))
			/* Deduplicate */
			(reduce inner-refs (lambda (acc rp)
				(if (has? acc rp) acc (merge acc (list rp))))
				'())))))

/* qpl-retarget-refs — rewrite (get_column tv ti col ci) where tv ∈ from-aliases
to (get_column to-alias false col false). Used to retarget inner refs in the
correlated WHERE conjunct to use the wrapping derived's alias after the cols
get projected through as passthroughs. */
(define qpl-retarget-refs (lambda (expr from-aliases to-alias)
	(match expr
		'((symbol get_column) tv ti col ci)
			(if (has? from-aliases tv)
				(list (quote get_column) to-alias false col false)
				expr)
		'((quote get_column) tv ti col ci)
			(if (has? from-aliases tv)
				(list (quote get_column) to-alias false col false)
				expr)
		(cons head args)
			(cons head (map (coalesceNil args '())
				(lambda (a) (qpl-retarget-refs a from-aliases to-alias))))
		expr)))

(define qpl-rewrite-correlated-limit-in-expr (lambda (expr)
	(match expr
		(cons sym args) (begin
			(define is-scalar (match sym
				(symbol inner_select) true
				(quote inner_select)  true
				'(quote inner_select) true
				'inner_select         true
				false))
			(if (and is-scalar (equal? (count args) 1))
				(list sym (qpl-rewrite-correlated-limit-with-rownumber (nth args 0)))
				(cons sym (map (coalesceNil args '())
					(lambda (a) (qpl-rewrite-correlated-limit-in-expr a))))))
		expr)))

(define qpl-rewrite-correlated-limit-tuple (lambda (t)
	(qpp-apply-to-tuple t qpl-rewrite-correlated-limit-in-expr)))

(define qpl-and-cond (lambda (a b)
	(if (or (nil? a) (equal? a true)) b
		(if (or (nil? b) (equal? b true)) a
			(list (quote and) a b)))))

/* qpl-outer-aliases — derive the visible table aliases of a 7-tuple's
table list. Used by qpl-rewrite-in-exists-tuple to qualify nil-tv outer
references that get moved into a synthesized inner WHERE (IN-rewrite).

Without qualification, a nil-tv ref `(get_column nil ID …)` placed into the
inner sub's WHERE re-resolves in inner scope (wrong) instead of outer
(correct). Qualifying with the outer's alias makes qpu-collect-outer-refs
detect it as a true outer-ref and lets cclasses/substitution handle it. */
(define qpl-outer-aliases (lambda (tables) (begin
	(define raw (map (coalesceNil tables '()) (lambda (td)
		(if (or (nil? td) (< (count td) 1)) nil
			(if (nil? (nth td 0))
				(if (>= (count td) 3) (nth td 2) nil)
				(nth td 0))))))
	(filter raw (lambda (a) (not (nil? a)))))))

/* qpl-qualify-walk — recursive walker for qpl-qualify-outer-nil-refs. */
(define qpl-qualify-walk (lambda (expr alias)
	(match expr
		'((symbol get_column) tv ti col ci)
			(if (nil? tv)
				(list (quote get_column) alias ti col ci)
				expr)
		'((quote get_column)  tv ti col ci)
			(if (nil? tv)
				(list (quote get_column) alias ti col ci)
				expr)
		(cons sym args)
			(cons sym (map (coalesceNil args '())
				(lambda (a) (qpl-qualify-walk a alias))))
		expr)))

/* qpl-qualify-outer-nil-refs — walks an expression tree and rewrites every
unqualified `(get_column nil ti col ci)` into `(get_column alias ti col ci)`
where `alias` is the SINGLE outer alias passed in. If outer-aliases has
0 or >1 entries, returns expr unchanged (the multi-outer-table case needs
schema info to disambiguate — a separate concern). */
(define qpl-qualify-outer-nil-refs (lambda (expr outer-aliases) (begin
	(if (not (equal? (count outer-aliases) 1)) expr
		(begin
			(define alias (nth outer-aliases 0))
			(qpl-qualify-walk expr alias))))))

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
				/* Qualify the sub's projection expression: nil-tv refs in the
				   inner SELECT field must resolve to the sub's own tables, not
				   leak as ambiguous refs into the synthesized equality. With
				   schemas threaded later this becomes unnecessary, but for
				   now (schemas=empty in column_resolve_pass), do the local
				   qualification here. */
				(define sub-aliases (qpl-outer-aliases (qpp-tuple-tables sub)))
				(define raw-sub-expr (nth (nth sub-fields 0) 1))
				(define sub-expr (qpl-qualify-outer-nil-refs raw-sub-expr sub-aliases))
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
COALESCE-COUNT > 0 boolean shape per FAQ §11.

The COALESCE around the inner_select is supplied by qpl-substitute-markers
via qpl-wrap-with-aggregate-neutral (FAQ §33) — for COUNT-LIKE aggregates
it auto-wraps. So this helper just emits `(> (inner_select count-sub) 0)`;
substitute-markers turns the inner_select into
`(coalesce (get_column sq_N false value false) 0)`, yielding the
mathematically equivalent `(> (coalesce sq-ref 0) 0)`.

If qpl-wrap-with-aggregate-neutral is later changed to NOT auto-wrap by
default, restore the explicit `(coalesce … 0)` here. */
(define qpl-wrap-as-count-gt-zero (lambda (count-sub)
	(list (quote >)
		(list (quote inner_select) count-sub)
		0)))

/* qpl-union-all-parts — if `sub` is a union_all form, return its branches
list; else nil. The parser emits UNION ALL as
  (union_all branches order limit offset)
where branches is a list of inner queries (each a 7-tuple OR another
union_all). For the FAQ §14 IN/EXISTS rewrite we ignore order/limit/offset
on the union (those would need additional handling; current callers only
hit branches+no-order-limit shapes). Returns nil if sub is a 7-tuple or
anything else. */
(define qpl-union-all-parts (lambda (sub) (match sub
	'(union_all branches order limit offset)
		(if (and (or (nil? order) (equal? order '()))
				 (nil? limit) (nil? offset))
			branches nil)
	'((symbol union_all) branches order limit offset)
		(if (and (or (nil? order) (equal? order '()))
				 (nil? limit) (nil? offset))
			branches nil)
	'((quote union_all) branches order limit offset)
		(if (and (or (nil? order) (equal? order '()))
				 (nil? limit) (nil? offset))
			branches nil)
	nil)))

/* qpl-and-or-from-branches — build `(or BR1 BR2 ...)` for IN-UNION rewrite
(positive: any branch matches) or `(and BR1 BR2 ...)` for NOT-IN rewrite
(negative: every branch fails). Empty branches: false for or, true for and. */
(define qpl-or-from-list (lambda (terms)
	(if (equal? (count terms) 0) false
		(if (equal? (count terms) 1) (nth terms 0)
			(reduce (cdr terms) (lambda (acc t)
				(list (quote or) acc t)) (car terms))))))

/* qpl-rewrite-in-exists — walk an expression tree, rewrite every
inner_select_in / inner_select_exists into the COALESCE-COUNT > 0 form.
Leaves scalar inner_select untouched (it's already in the form the
substitution walker expects).

FAQ §14: IN/EXISTS over UNION ALL is rewritten as OR-of-IN/EXISTS per
branch BEFORE the COUNT lowering, so each branch becomes its own
COALESCE(COUNT(),0)>0 wrapper. Equivalent semantics, gives the unnest
a flat set of 7-tuple subs to process.

`outer-aliases` is the list of OUTER table aliases visible at the SQL scope
of this expression — passed through so qpl-make-count-subquery-for-in can
qualify nil-tv outer references in `a` before placing them inside the inner
sub's WHERE. */
(define qpl-rewrite-in-exists (lambda (expr outer-aliases) (begin
	(define k (qpl-marker-kind expr))
	(if (equal? k (quote inner_select_in))
		(begin
			(define sub (qpl-marker-subquery expr))
			(define branches (qpl-union-all-parts sub))
			(if (not (nil? branches))
				/* IN (UNION ALL of branches) → (lhs IN br1) OR (lhs IN br2) OR …
				   Re-run rewrite on each new marker to recurse properly. */
				(qpl-rewrite-in-exists
					(qpl-or-from-list (map branches (lambda (br)
						(list (quote inner_select_in) (qpl-marker-lhs expr) br))))
					outer-aliases)
				(qpl-wrap-as-count-gt-zero
					(qpl-make-count-subquery-for-in
						(qpl-qualify-outer-nil-refs
							(qpl-rewrite-in-exists (qpl-marker-lhs expr) outer-aliases)
							outer-aliases)
						sub))))
		(if (equal? k (quote inner_select_exists))
			(begin
				(define sub (qpl-marker-subquery expr))
				(define branches (qpl-union-all-parts sub))
				(if (not (nil? branches))
					/* EXISTS (UNION ALL of branches) → EXISTS br1 OR EXISTS br2 OR … */
					(qpl-rewrite-in-exists
						(qpl-or-from-list (map branches (lambda (br)
							(list (quote inner_select_exists) br))))
						outer-aliases)
					(qpl-wrap-as-count-gt-zero
						(qpl-make-count-subquery-for-exists sub))))
			(match expr
				(cons sym args) (cons sym (map (coalesceNil args '())
					(lambda (a) (qpl-rewrite-in-exists a outer-aliases))))
				expr))))))

/* qpl-rewrite-in-exists-fields — apply the rewrite to each projection. */
(define qpl-rewrite-in-exists-fields (lambda (fields outer-aliases)
	(map (coalesceNil fields '()) (lambda (pair) (match pair
		'(name expr) (list name (qpl-rewrite-in-exists expr outer-aliases))
		pair)))))

(define qpl-rewrite-in-exists-group (lambda (group outer-aliases)
	(if (nil? group) nil
		(map group (lambda (e) (qpl-rewrite-in-exists e outer-aliases))))))

(define qpl-rewrite-in-exists-order (lambda (order outer-aliases)
	(if (nil? order) nil
		(map order (lambda (item) (match item
			'(expr dir) (list (qpl-rewrite-in-exists expr outer-aliases) dir)
			item))))))

/* qpl-rewrite-in-exists-tuple — apply qpl-rewrite-in-exists to every
expression slot of a 7-tuple. Derives outer-aliases from the tuple's tables
and threads them through so synthesized IN-equality predicates get qualified
correctly. */
(define qpl-rewrite-in-exists-tuple (lambda (t) (begin
	(define outer-aliases (qpl-outer-aliases (qpp-tuple-tables t)))
	(qpp-rebuild-tuple
		(qpp-tuple-schema t)
		(qpp-tuple-tables t)
		(qpl-rewrite-in-exists-fields (qpp-tuple-fields t) outer-aliases)
		(qpl-rewrite-in-exists (qpp-tuple-condition t) outer-aliases)
		(qpl-rewrite-in-exists-group (qpp-tuple-group t) outer-aliases)
		(qpl-rewrite-in-exists (qpp-tuple-having t) outer-aliases)
		(qpl-rewrite-in-exists-order (qpp-tuple-order t) outer-aliases)
		(qpp-tuple-limit t)
		(qpp-tuple-offset t)))))

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

/* qpl-sub-aggregate-neutral — if sub is a single-aggregate-field 7-tuple,
return the aggregate's neutral element (e.g. 0 for COUNT, nil for SUM/MIN/
MAX). Else return nil. FAQ §33 static-group preservation: when an outer
binding has no matching inner row, the aggregate's NEUTRAL is the correct
value for that binding (e.g. COUNT(*) → 0, not NULL). LEFT JOIN semantics
gives NULL by default; COALESCE-with-neutral wrappers restore the
mathematically correct behaviour. Only meaningful for COUNT-like aggregates
where neutral ≠ NULL — SUM/MIN/MAX have NULL neutral so wrapping is
pointless (NULL coalesce NULL = NULL). */
(define qpl-sub-aggregate-neutral (lambda (sub)
	(if (not (qpp-tuple? sub)) nil
		(begin
			(define fields-pairs (qpp-fields-to-pairs (qpp-tuple-fields sub)))
			(if (not (equal? (count fields-pairs) 1)) nil
				(begin
					(define field-pair (nth fields-pairs 0))
					(define field-expr (nth field-pair 1))
					(define agg-info (match field-expr
						'((symbol aggregate) inner reducer neutral)
							(list inner reducer neutral)
						'((quote aggregate)  inner reducer neutral)
							(list inner reducer neutral)
						nil))
					(if (nil? agg-info) nil
						(begin
							(define inner (nth agg-info 0))
							(define neutral (nth agg-info 2))
							/* Skip COALESCE wrap when sub has HAVING — HAVING can
							   filter the group to empty, in which case SQL spec
							   says scalar = NULL. COALESCE-to-neutral would convert
							   NULL to 0 for COUNT — wrong per SQL. */
							(define sub-having (qpp-tuple-having sub))
							(if (and (not (nil? sub-having)) (not (equal? sub-having true)))
								nil
							/* Only return the neutral for COUNT-LIKE aggregates
							   where the inner expression yields 0/1 (non-NULL):
							     COUNT(*)    inner = 1
							     COUNT(expr) inner = (if (nil? expr) 0 1)
							   For SUM/MIN/MAX the inner can yield NULL and the
							   SQL semantic for empty input is NULL (not the
							   reducer's algebraic neutral). Returning nil for
							   those cases means qpl-wrap-with-aggregate-neutral
							   does NOT add COALESCE. */
							(if (qpl-is-count-like-inner? inner)
								neutral
								nil))))))))))

/* qpl-is-count-like-inner? — true if the aggregate's `inner` expression is
the COUNT(*) constant 1 or the parser-emitted COUNT(expr) pattern
(if (nil? expr) 0 1). Other shapes (column refs, arithmetic, etc.) are
treated as SUM-like and yield NULL on empty input per SQL semantics. */
(define qpl-is-count-like-inner? (lambda (inner)
	(if (equal? inner 1) true
		(match inner
			'((symbol if) ((symbol nil?) _) 0 1) true
			'((quote if)  ((quote nil?)  _) 0 1) true
			'(if          (nil?           _) 0 1) true
			false))))

/* qpl-wrap-with-aggregate-neutral — given the sq-ref and a sub-tuple,
return COALESCE(sq-ref, neutral) if sub's aggregate has a non-nil neutral
(e.g. COUNT's 0); else return sq-ref as-is. */
(define qpl-wrap-with-aggregate-neutral (lambda (sq-ref sub) (begin
	(define neutral (qpl-sub-aggregate-neutral sub))
	/* `equal? 0 nil` is true in this Scheme dialect — see
	   feedback memory `equal? Bug`. Use explicit nil? only. */
	(if (nil? neutral)
		sq-ref
		(list (quote coalesce) sq-ref neutral)))))

/* qpl-substitute-markers — walks an expression. Each scalar inner_select
encountered is replaced by a get_column reference; the marker's subquery is
recorded into `acc` (a newsession with key "list" → list of (sq-alias subquery)).
Non-scalar markers (IN/EXISTS) trigger an error — those are Phase 3+.

Per FAQ §33 static-group preservation: when sub is a COUNT-LIKE aggregate
(inner = 1 or (if (nil? expr) 0 1)), wrap the sq-ref in COALESCE so empty
inner produces 0 instead of NULL (LEFT JOIN's NULL-extension). SUM/MIN/MAX
return NULL on empty per SQL semantics — those are NOT wrapped. */
(define qpl-substitute-markers (lambda (expr acc) (begin
	(define k (qpl-marker-kind expr))
	(if (equal? k (quote inner_select))
		(begin
			(define sub (qpl-marker-subquery expr))
			(define sq-alias (qpl-fresh-sq-alias))
			(acc "list" (merge (coalesceNil (acc "list") '())
				(list (list sq-alias sub))))
			(qpl-wrap-with-aggregate-neutral (qpl-sq-col sq-alias) sub))
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
NOTE: (nil? '()) is FALSE in this dialect — use (> (count ...) 0) instead.

Phase 5: also detect COMPOUND expressions that CONTAIN aggregates (e.g.
AVG = (/ SUM COUNT), `MAX-MIN`, `SUM(x)+1`, etc.). These need multi-
aggregate decomposition into qpir-groupby + qpir-map. */
(define qpl-needs-decompose? (lambda (sub)
	(or
		(> (count (qpl-collect-aggregates-in-fields (qpp-tuple-fields sub))) 0)
		(> (count (coalesceNil (qpp-tuple-group sub) '())) 0)
		(qpl-fields-contain-compound-aggregate? (qpp-tuple-fields sub)))))

/* qpl-fields-contain-compound-aggregate? — true if any field's expression
contains an aggregate inside a non-bare position (e.g. (/ agg agg), (+ agg 1)). */
(define qpl-fields-contain-compound-aggregate? (lambda (fields)
	(reduce (qpp-fields-to-pairs fields) (lambda (acc pair) (match pair
		'(name expr) (or acc
			(and (not (qpl-is-aggregate-expr? expr))
				 (qpl-expr-has-aggregate? expr)))
		acc)) false)))

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
			/* HAVING passes through into the qpir-groupby's having slot. */
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
					/* Correlation gate: decomposition into qpir-groupby is
					   only required when the sub references outer scope —
					   that's when unnest §3.3 needs operator-level groupby
					   to push the dep-join through. UNCORRELATED subs lower
					   cleanly as a passthrough qpir-leaf; legacy
					   build_queryplan_inner handles the full 7-tuple's
					   GROUP/HAVING/ORDER/LIMIT slots in one shot.

					   This unblocks GROUP-BY-with-ORDER-BY-and-LIMIT
					   scalar subqueries that decompose would error on
					   ("no aggregates found" when fields are pure
					   group-key projections). */
					(if (qpl-tuple-has-outer-refs? leaf-tuple)
						/* Correlated: existing decompose-or-hoist logic */
						(if (qpl-needs-decompose? leaf-tuple)
							(qpl-build-groupby-wrapped-inner leaf-tuple)
							(qpl-build-simple-leaf-inner leaf-tuple))
						/* Uncorrelated: pass full 7-tuple through as-is.
						   Field is already renamed to "value" so callers
						   reference sq.value uniformly. */
						(qpir-leaf leaf-tuple)))
				lifted)))))

/* qpl-tuple-own-aliases — return the list of table aliases the tuple itself
introduces (its tables list). Used by qpl-tuple-has-outer-refs?. */
(define qpl-tuple-own-aliases (lambda (sub)
	(map (coalesceNil (qpp-tuple-tables sub) '()) (lambda (td)
		(if (or (nil? td) (< (count td) 1)) nil (nth td 0))))))

/* qpl-tuple-has-outer-refs? — true when any expression in the sub-tuple
(fields, condition, group, having, order) references a table-alias that is
NOT in the sub's own tables list. This is the correlation predicate used to
decide whether the sub needs operator-level decomposition for unnest §3.3
push-down. UNcorrelated subqueries can pass through as a qpir-leaf with
the full 7-tuple intact; legacy build_queryplan_inner handles their
WHERE/GROUP/HAVING/ORDER/LIMIT slots directly. */
(define qpl-tuple-has-outer-refs? (lambda (sub) (begin
	(define own-aliases (qpl-tuple-own-aliases sub))
	(define field-refs (qpir-assoc-list-refs
		(qpp-fields-to-pairs (qpp-tuple-fields sub))))
	(define cond-refs (qpir-expr-column-refs
		(coalesceNil (qpp-tuple-condition sub) true)))
	(define group-refs (qpir-expr-list-refs
		(coalesceNil (qpp-tuple-group sub) '())))
	(define having-refs (qpir-expr-column-refs
		(coalesceNil (qpp-tuple-having sub) true)))
	(define order-refs (qpir-order-list-refs
		(coalesceNil (qpp-tuple-order sub) '())))
	(define all-refs (merge (merge (merge (merge
		field-refs cond-refs) group-refs) having-refs) order-refs))
	(define outer-refs (filter all-refs (lambda (ref) (match ref
		'(tv col) (and (not (nil? tv)) (not (has? own-aliases tv)))
		false))))
	(> (count outer-refs) 0))))

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

/* qpl-agg-counter — counter for synthesized aggregate names in decomposition. */
(define qpl-agg-counter (newsession))
(qpl-agg-counter "n" 0)
(define qpl-fresh-agg-name (lambda () (begin
	(qpl-agg-counter "n" (+ (qpl-agg-counter "n") 1))
	(concat "__agg_" (string (qpl-agg-counter "n"))))))

/* qpl-extract-aggregates — walk an expression; for each bare (aggregate …)
subexpr, allocate a fresh __agg_N name and replace the subexpr with a
(get_column nil false __agg_N false) placeholder. Accumulate (name agg-expr)
pairs into agg-acc. Returns the rewritten expression.

The placeholder uses nil tv (the qpir-map projection won't qualify it; the
SUB's outer scope resolves it via name match against the agg-projections
that the qpir-groupby outputs).

This is the FAQ §33 compound-aggregate decomposition: split user-written
(/ SUM COUNT) (AVG) or (- MAX MIN) (range) into N bare aggregates + a final
projection that COMBINES them. */
(define qpl-extract-aggregates (lambda (expr agg-acc)
	(if (qpl-is-aggregate-expr? expr)
		(begin
			(define n (qpl-fresh-agg-name))
			(agg-acc "list" (merge (coalesceNil (agg-acc "list") '())
				(list (list n expr))))
			(list (quote get_column) nil false n false))
		(match expr
			(cons head args) (cons head (map (coalesceNil args '())
				(lambda (a) (qpl-extract-aggregates a agg-acc))))
			expr))))

(define qpl-build-groupby-wrapped-inner (lambda (sub) (begin
	(define fields-pairs (qpp-fields-to-pairs (qpp-tuple-fields sub)))
	(if (not (equal? (count fields-pairs) 1))
		(error (concat "qpl-build-groupby-wrapped-inner: expected 1 field, found "
			(string (count fields-pairs)) " — multi-field inner subqueries are phase 5+")) nil)
	(define field-pair (nth fields-pairs 0))
	(define field-expr (nth field-pair 1))
	/* Decompose field-expr into N bare aggregates + final projection.
	   For bare aggregate: agg-acc gets 1 entry, final-expr is a get_column
	   placeholder. For compound (e.g. AVG): N entries, final-expr is the
	   compound with placeholder refs. */
	(define agg-acc (newsession))
	(agg-acc "list" '())
	(define final-expr (qpl-extract-aggregates field-expr agg-acc))
	(define agg-pairs (agg-acc "list"))
	(if (equal? (count agg-pairs) 0)
		(error "qpl-build-groupby-wrapped-inner: no aggregates found — shouldn't reach here") nil)

	/* Collect every column ref needed by the aggregates' inner expressions
	   + WHERE; the leaf must project all of them so runtime sees them. */
	(define agg-inners (map agg-pairs (lambda (pair) (begin
		(define agg-expr (nth pair 1))
		(define agg-args (cdr agg-expr))
		(nth agg-args 0)))))
	(define leaf-cols-from-aggs (reduce agg-inners (lambda (acc inner)
		(merge acc (qpir-expr-column-refs inner))) '()))
	(define leaf-cols-from-where
		(qpir-expr-column-refs (coalesceNil (qpp-tuple-condition sub) true)))
	(define leaf-cols (qpl-dedupe-col-refs (merge leaf-cols-from-aggs leaf-cols-from-where)))
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
		'()
		nil
		'()
		nil
		nil)))
	(define leaf-with-select (qpl-wrap-with-select-if-needed
		(qpp-tuple-condition sub)
		leaf-bare))
	(define group-keys (coalesceNil (qpp-tuple-group sub) '()))
	(define groupby (qpir-groupby
		group-keys
		agg-pairs       /* N aggregates with synthesized __agg_N names */
		(qpp-tuple-having sub)
		leaf-with-select))
	(if (equal? (count agg-pairs) 1)
		/* Single bare aggregate: legacy path — qpir-groupby outputs "value"
		   directly. The single agg gets renamed to "value" so callers
		   reference sq.value uniformly. */
		(qpir-groupby
			group-keys
			(list (list "value" (nth (nth agg-pairs 0) 1)))
			(qpp-tuple-having sub)
			leaf-with-select)
		/* Multi-aggregate: wrap with qpir-map that projects "value" = final-expr.
		   final-expr references the synthesized agg names via get_column nil
		   placeholders; downstream lower wraps the groupby as a derived table
		   and the placeholders resolve to the derived's columns. */
		(qpir-map (list (list "value" final-expr)) groupby)))))

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
			/* Step 0a — drop redundant LIMIT k from sub-tuples whose WHERE
			   equi-binds every outer-ref to an inner column (FAQ §38/§39
			   simple unnesting + trivial dep-join). Without this, the LIMIT
			   becomes a global LIMIT on the derived sub and clips correlated
			   rows before the join. */
			(define t-lim (qpl-rewrite-redundant-limit-tuple t))
			/* Step 0b — FAQ §43 ROW_NUMBER PARTITION rewrite per FAQ:
			   "rewrite the inner select into a window over the same domain
			   keys: SELECT *, ROW_NUMBER() OVER (PARTITION BY <outer-refs>
			   ORDER BY x) rn ... WHERE rn BETWEEN o+1 AND k+o".
			   ENABLED 2026-06-01 with FAQ §35 canonical-names refinement:
			   PARTITION BY uses inner-equibound col where available,
			   avoiding correlated PARTITION BY that legacy can't handle. */
			(define t-rn (qpl-rewrite-correlated-limit-tuple t-lim))
			(define t-prime (qpl-rewrite-in-exists-tuple t-rn))
			(if (not (qpl-tuple-has-markers? t-prime))
				(qpir-leaf t-prime)
				(qpl-lift-with-markers t-prime))))))

(define qpl-lift-with-markers (lambda (t) (begin
	/* Reject shapes Phase 2 does not yet handle: group-by markers,
	   order-by markers. HAVING-level markers are supported via the
	   same substitution mechanism as fields/condition — the substituted
	   HAVING references sq_X.value which resolves through the dep-join
	   chain wrapped around the outer-leaf. */
	(if (> (reduce (coalesceNil (qpp-tuple-group t) '()) (lambda (acc e)
			(+ acc (count (qpl-collect-markers e)))) 0) 0)
		(error "lift_dep_joins_pass: GROUP-BY-level marker not yet supported (Phase 3+)") nil)
	(if (> (reduce (coalesceNil (qpp-tuple-order t) '()) (lambda (acc item) (match item
			'(expr dir) (+ acc (count (qpl-collect-markers expr)))
			acc)) 0) 0)
		(error "lift_dep_joins_pass: ORDER-BY-level marker not yet supported (Phase 3+)") nil)

	/* Collect + substitute all scalar markers from fields, condition, and
	   HAVING. Order matters: fields → condition → having for deterministic
	   sq_N numbering. */
	(define acc (newsession))
	(acc "list" '())
	(define orig-fields (qpp-tuple-fields t))
	(define sub-fields (qpl-substitute-fields orig-fields acc))
	(define orig-cond (qpp-tuple-condition t))
	(define sub-cond (qpl-substitute-markers orig-cond acc))
	(define orig-having (qpp-tuple-having t))
	(define sub-having
		(if (nil? orig-having) nil
			(qpl-substitute-markers orig-having acc)))
	(define markers (acc "list"))

	(if (equal? (count markers) 0)
		/* Nothing actually got lifted (defensive: should not happen because
		   qpl-tuple-has-markers? said yes). Fall back to leaf wrap. */
		(qpir-leaf t)
		(begin
			/* Build outer leaf: substituted fields, WHERE replaced by true
			   (real condition is re-applied above the dep-join chain so its
			   sq.value references can resolve to the dep-join's right side).
			   HAVING uses the substituted form directly — the sq_X.value
			   refs in HAVING resolve through the dep-join chain just like
			   refs in fields do (both are at the outer-leaf's projection level). */
			(define outer-leaf (qpir-leaf (qpp-rebuild-tuple
				(qpp-tuple-schema t)
				(qpp-tuple-tables t)
				sub-fields
				true
				(qpp-tuple-group t)
				sub-having
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
