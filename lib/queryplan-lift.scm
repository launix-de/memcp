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

/* qpl-marker-kind — return the inner_select kind symbol of expr, or nil.
Use explicit head+arity checks instead of match patterns: in this Scheme
matcher, unquoted symbols in list patterns can bind, which made ordinary
2/3-element expressions look like subquery markers. */
(define qpl-marker-head-name (lambda (head) (serialize head)))
(define qpl-marker-head-eq? (lambda (head name) (begin
	(define s (qpl-marker-head-name head))
	(or
		(equal? s name)
		(equal? s (concat "(quote " name ")"))
		(equal? s (concat "(symbol " name ")"))))))
(define qpl-marker-kind (lambda (expr)
	(if (or (nil? expr) (not (list? expr)) (equal? (count expr) 0))
		nil
		(begin
			(define head (nth expr 0))
			(if (and (equal? (count expr) 2)
				(qpl-marker-head-eq? head "inner_select"))
				(quote inner_select)
				(if (and (equal? (count expr) 3)
					(qpl-marker-head-eq? head "inner_select_in"))
					(quote inner_select_in)
					(if (and (equal? (count expr) 2)
						(qpl-marker-head-eq? head "inner_select_exists"))
						(quote inner_select_exists)
						nil)))))))

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
		(cons head args) (if (and (is-eq-sym head) (list? args) (equal? (count args) 2))
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

(define qpl-inner-refs-bound-to-ref (lambda (ref-pair conjuncts inner-aliases) (begin
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
	(merge_unique (filter (map conjuncts (lambda (c) (match c
		(cons head args) (if (and (is-eq-sym head) (list? args) (equal? (count args) 2))
			(begin
				(define a (nth args 0))
				(define b (nth args 1))
				(define ka (ref-key a))
				(define kb (ref-key b))
				(if (or (nil? ka) (nil? kb)) nil
					(if (and (equal? ka ref-pair) (has? inner-aliases (nth kb 0)))
						kb
						(if (and (equal? kb ref-pair) (has? inner-aliases (nth ka 0)))
							ka
							nil))))
			nil)
		nil)))
		(lambda (ref) (not (nil? ref))))))))

(define qpl-table-ref-primary? (lambda (td tv col)
	(match td
		'(alias tschema ttbl _ _)
		(if (not (equal? alias tv)) false
			(begin
				(define base_tbl (planner_table_source_base ttbl))
				(if (not (string? base_tbl)) false
					(begin
						(define cols (try (lambda () (get_schema tschema base_tbl))
							(lambda (e) nil)))
						(reduce (coalesceNil cols '()) (lambda (col_found coldef)
							(or col_found
								(and
									(equal?? (coldef "Field") col)
									(equal? (coldef "Key") "PRI"))))
							false)))))
		_ false)))

(define qpl-inner-ref-is-primary? (lambda (tables ref-pair)
	(match ref-pair
		'(tv col)
		(reduce tables (lambda (found td)
			(or found (qpl-table-ref-primary? td tv col)))
			false)
		_ false)))

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
returns sub unchanged. The dropped-LIMIT case keeps a hidden marker column so
the later scalar-cardinality pass does not mistake it for an original no-LIMIT
subquery. */
(define qpl-drop-redundant-correlated-limit (lambda (sub)
	(if (not (qpp-tuple? sub)) sub
		(begin
			(define lim (qpp-tuple-limit sub))
			(define off (qpp-tuple-offset sub))
			(define ord (qpp-tuple-order sub))
			(define has_aggregate_field
				(reduce (qpp-fields-to-pairs (coalesceNil (qpp-tuple-fields sub) '()))
					(lambda (acc pair) (match pair
						'(_ expr) (or acc (qpl-expr-has-aggregate? expr))
						acc))
					false))
			/* Drop ONLY when: LIMIT is set, no OFFSET, no ORDER BY.
			ORDER BY means the LIMIT selects a SPECIFIC subset (e.g. top-k);
			dropping the LIMIT would change which rows are returned, even
			if equi-binding bounds the cardinality. The ROW_NUMBER PARTITION
			rewrite (qpl-rewrite-correlated-limit-with-rownumber) handles
			the ordered case correctly.

			Nested markers do not by themselves affect this cardinality proof:
			if this subquery level is equi-bound through a unique inner key,
			it can emit at most one row per outer binding before evaluating
			nested scalar/IN expressions. Keep those markers in place and only
			remove the redundant LIMIT. */
			(if (or (nil? lim) (not (nil? off))
				(and (not (nil? ord)) (> (count ord) 0))) sub
				(begin
					(define inner-aliases (qpl-outer-aliases (qpp-tuple-tables sub)))
					(define cond (qpp-tuple-condition sub))
					(define conjuncts (qpl-and-conjuncts cond))
					/* All column refs in the WHERE that are NOT bound by an inner
					alias are outer-refs. We require EVERY such outer-ref to be
					equi-bound to an inner column. */
					/* Only inspect refs owned by this subquery level. Nested
					inner_select / IN / EXISTS markers have their own LIMIT
					semantics; including their refs here falsely prevents the
					equi-bound LIMIT drop and routes simple LIMIT 1 chains into
					the ROW_NUMBER window fallback. */
					(define all-refs (qpl-extract-col-refs-skip-nested cond))
					(define outer-refs (filter all-refs (lambda (rp) (match rp
						'(tv col) (and (not (nil? tv)) (not (has? inner-aliases tv)))
						false))))
					(if (equal? (count outer-refs) 0) sub
						(begin
							(define all-bound (reduce outer-refs (lambda (acc ref)
								(and acc (qpl-ref-bound-by-equality? ref conjuncts inner-aliases)))
								true))
							(define all-bound-to-primary (reduce outer-refs (lambda (acc ref)
								(and acc
									(begin
										(define inner_refs (qpl-inner-refs-bound-to-ref ref conjuncts inner-aliases))
										(reduce inner_refs (lambda (found inner_ref)
											(or found (qpl-inner-ref-is-primary? (qpp-tuple-tables sub) inner_ref)))
											false))))
								true))
							(define any-bound-to-primary (reduce outer-refs (lambda (acc ref)
								(or acc
									(begin
										(define inner_refs (qpl-inner-refs-bound-to-ref ref conjuncts inner-aliases))
										(reduce inner_refs (lambda (found inner_ref)
											(or found (qpl-inner-ref-is-primary? (qpp-tuple-tables sub) inner_ref)))
											false))))
								false))
							(define is-eq-conj (lambda (c) (match c
								(cons head args) (and (list? args) (equal? (count args) 2)
									(match head
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
										false))
								false)))
							(define any-primary-equality (reduce conjuncts (lambda (acc c)
								(or acc
									(and (is-eq-conj c)
										(reduce (qpl-extract-col-refs c) (lambda (found ref)
											(or found
												(match ref
													'(tv _)
													(and
														(has? inner-aliases tv)
														(qpl-inner-ref-is-primary? (qpp-tuple-tables sub) ref))
													false)))
											false))))
								false))
							(if (or (and all-bound has_aggregate_field) any-bound-to-primary any-primary-equality)
								(qpp-rebuild-tuple
									(qpp-tuple-schema sub)
									(qpp-tuple-tables sub)
									(merge
										(qpp-fields-to-pairs (qpp-tuple-fields sub))
										(list (list "__qpl_dropped_limit" lim)))
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
			(if (and is-scalar (list? args) (equal? (count args) 1))
				(list sym (qpl-drop-redundant-correlated-limit
					(qpp-apply-to-tuple (nth args 0)
						qpl-rewrite-redundant-limit-in-expr)))
				(if (list? args)
					(cons sym (map args
						(lambda (a) (qpl-rewrite-redundant-limit-in-expr a))))
					expr)))
		_ expr)))

(define qpl-rewrite-redundant-limit-tuple (lambda (t)
	(qpp-apply-to-tuple t qpl-rewrite-redundant-limit-in-expr)))

(define qpl-predicate-binds-limited-leaf-primary? (lambda (pred leaf-tuple)
	(begin
		(define leaf-aliases (qpl-outer-aliases (qpp-tuple-tables leaf-tuple)))
		(define eq-conj? (lambda (c) (match c
			(cons head args) (and (list? args) (equal? (count args) 2)
				(match head
					(symbol equal??) true
					(quote equal??) true
					'(quote equal??) true
					'equal?? true
					(symbol equal?) true
					(quote equal?) true
					'(quote equal?) true
					'equal? true
					(symbol =) true
					(quote =) true
					'(quote =) true
					'= true
					false))
			false)))
		(define unwrap-truthy (lambda (c) (match c
			'((symbol sql_truthy) inner) inner
			'((quote sql_truthy) inner) inner
			_ c)))
		(define col-ref (lambda (e) (match e
			'((symbol get_column) tv _ col _) (list tv col)
			'((quote get_column) tv _ col _) (list tv col)
			nil)))
		(define binds-primary? (lambda (inner-ref outer-ref)
			(match inner-ref
				'(tv _col)
				(and
					(has? leaf-aliases tv)
					(not (nil? outer-ref))
					(match outer-ref
						'(otv _ocol) (not (has? leaf-aliases otv))
						false)
					(qpl-inner-ref-is-primary? (qpp-tuple-tables leaf-tuple) inner-ref))
				false)))
		(reduce (qpl-and-conjuncts pred) (lambda (found conj)
			(or found
				(begin
					(define eq-conj (unwrap-truthy conj))
					(and (eq-conj? eq-conj)
						(begin
							(define lhs (nth eq-conj 1))
							(define rhs (nth eq-conj 2))
							(define lref (col-ref lhs))
							(define rref (col-ref rhs))
							(or
								(binds-primary? lref rref)
								(binds-primary? rref lref)))))))
			false))))

(define qpl-drop-limited-leaf-limit (lambda (leaf-tuple)
	(qpp-rebuild-tuple
		(qpp-tuple-schema leaf-tuple)
		(qpp-tuple-tables leaf-tuple)
		(merge
			(qpp-fields-to-pairs (qpp-tuple-fields leaf-tuple))
			(list (list "__qpl_dropped_limit" (qpp-tuple-limit leaf-tuple))))
		(qpp-tuple-condition leaf-tuple)
		(qpp-tuple-group leaf-tuple)
		(qpp-tuple-having leaf-tuple)
		(qpp-tuple-order leaf-tuple)
		nil
		nil)))

(define qpl-drop-redundant-limits-under-select (lambda (pred node)
	(match (qpir-kind node)
		(quote qpir-leaf) (begin
			(define leaf-tuple (qpir-leaf-7tuple node))
			(if (and
				(not (nil? (qpp-tuple-limit leaf-tuple)))
				(nil? (qpp-tuple-offset leaf-tuple))
				(or (nil? (qpp-tuple-order leaf-tuple))
					(equal? (qpp-tuple-order leaf-tuple) '()))
				(qpl-predicate-binds-limited-leaf-primary? pred leaf-tuple))
				(qpir-leaf (qpl-drop-limited-leaf-limit leaf-tuple))
				node))
		(quote qpir-dep-join)
		(qpir-dep-join
			(qpir-dep-join-predicate node)
			(qpl-drop-redundant-limits-under-select pred (qpir-dep-join-left node))
			(qpl-drop-redundant-limits-under-select pred (qpir-dep-join-right node))
			(qpir-dep-join-accessing node)
			(qpir-dep-join-rhs-alias node))
		(quote qpir-join)
		(qpir-join
			(qpir-join-type node)
			(qpir-join-predicate node)
			(qpl-drop-redundant-limits-under-select pred (qpir-join-left node))
			(qpl-drop-redundant-limits-under-select pred (qpir-join-right node))
			(qpir-join-rhs-alias node))
		_ node)))

(define qpl-drop-redundant-qpir-limits (lambda (node)
	(match (qpir-kind node)
		(quote qpir-select) (begin
			(define child (qpl-drop-redundant-qpir-limits (qpir-select-child node)))
			(qpir-select
				(qpir-select-predicate node)
				(qpl-drop-redundant-limits-under-select
					(qpir-select-predicate node)
					child)))
		(quote qpir-map)
		(qpir-map (qpir-map-projections node)
			(qpl-drop-redundant-qpir-limits (qpir-map-child node)))
		(quote qpir-groupby)
		(qpir-groupby (qpir-groupby-keys node) (qpir-groupby-aggs node)
			(qpir-groupby-having node)
			(qpl-drop-redundant-qpir-limits (qpir-groupby-child node)))
		(quote qpir-window)
		(qpir-window (qpir-window-partition node) (qpir-window-order node)
			(qpir-window-computations node)
			(qpl-drop-redundant-qpir-limits (qpir-window-child node)))
		(quote qpir-join)
		(qpir-join (qpir-join-type node) (qpir-join-predicate node)
			(qpl-drop-redundant-qpir-limits (qpir-join-left node))
			(qpl-drop-redundant-qpir-limits (qpir-join-right node))
			(qpir-join-rhs-alias node))
		(quote qpir-dep-join)
		(qpir-dep-join (qpir-dep-join-predicate node)
			(qpl-drop-redundant-qpir-limits (qpir-dep-join-left node))
			(qpl-drop-redundant-qpir-limits (qpir-dep-join-right node))
			(qpir-dep-join-accessing node)
			(qpir-dep-join-rhs-alias node))
		(quote qpir-union)
		(qpir-union (qpir-union-order node) (qpir-union-limit node)
			(qpir-union-offset node)
			(map (qpir-union-branches node) qpl-drop-redundant-qpir-limits))
		(quote qpir-iterate)
		(qpir-iterate
			(qpl-drop-redundant-qpir-limits (qpir-iterate-seed node))
			(qpl-drop-redundant-qpir-limits (qpir-iterate-recursive node))
			(qpir-iterate-iterationscans node))
		_ node)))

/* ==================== FAQ §43 ROW_NUMBER PARTITION rewrite ==================== */

/* qpl-static-aggregate-limit-redundant? — true for scalar aggregate
subqueries without GROUP BY. Such a query already emits at most one logical
row per domain binding, so LIMIT 1 is a no-op. Keeping the LIMIT would send
the subquery through the ROW_NUMBER rewrite and incorrectly move correlated
WHERE predicates outside the aggregate input. */
(define qpl-static-aggregate-limit-redundant? (lambda (sub)
	(and
		(equal? (coalesceNil (qpp-tuple-group sub) '()) '())
		(reduce (qpp-fields-to-pairs (qpp-tuple-fields sub)) (lambda (acc pair)
			(match pair
				'(_ expr) (or acc (qpl-expr-has-aggregate? expr))
				acc)) false))))

(define qpl-drop-static-aggregate-limit (lambda (sub)
	(if (and
		(qpl-static-aggregate-limit-redundant? sub)
		(not (nil? (qpp-tuple-limit sub)))
		(nil? (qpp-tuple-offset sub))
		(or (nil? (qpp-tuple-order sub))
			(equal? (count (qpp-tuple-order sub)) 0)))
		(qpp-rebuild-tuple
			(qpp-tuple-schema sub)
			(qpp-tuple-tables sub)
			(qpp-tuple-fields sub)
			(qpp-tuple-condition sub)
			(qpp-tuple-group sub)
			(qpp-tuple-having sub)
			(qpp-tuple-order sub)
			nil
			nil)
		sub)))

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
				'((symbol sql_truthy) inner) (qpl-find-equibind-inner-col inner outer-ref inner-aliases)
				'((quote sql_truthy) inner) (qpl-find-equibind-inner-col inner outer-ref inner-aliases)
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
		(define single-inner-alias (if (equal? (count inner-aliases) 1) (car inner-aliases) nil))
		(define normalize-inner-info (lambda (info)
			(match info
				'(nil col) (if (nil? single-inner-alias) info (list single-inner-alias col))
				_ info)))
		(define out-tv (nth outer-ref 0))
		(define out-col (nth outer-ref 1))
		(if (and (not (nil? li))
			(equal? (nth li 0) out-tv) (equal? (nth li 1) out-col)
			(not (nil? ri))
			(has? inner-aliases (nth (normalize-inner-info ri) 0)))
			(normalize-inner-info ri)
			(if (and (not (nil? ri))
				(equal? (nth ri 0) out-tv) (equal? (nth ri 1) out-col)
				(not (nil? li))
				(has? inner-aliases (nth (normalize-inner-info li) 0)))
				(normalize-inner-info li)
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

(define qpl-limit-equibind-inner-cols (lambda (outer-refs where inner-aliases)
	(reduce outer-refs (lambda (acc rp)
		(match rp
			'(tv _col)
			(if (has? inner-aliases tv) acc
				(begin
					(define inner-equiv (qpl-find-equibind-inner-col where rp inner-aliases))
					(if (or (nil? inner-equiv) (has? acc inner-equiv))
						acc
						(merge acc (list inner-equiv)))))
			acc))
		'())))

(define qpl-limit-equibind-wrapper-condition (lambda (outer-refs where inner-aliases wrap-alias)
	(reduce outer-refs (lambda (acc rp) (match rp
		'(outer-tv outer-col)
		(if (has? inner-aliases outer-tv) acc
			(begin
				(define inner-equiv (qpl-find-equibind-inner-col where rp inner-aliases))
				(if (nil? inner-equiv) acc
					(qpl-and-cond acc
						(list (quote equal??)
							(list (quote get_column) wrap-alias false (nth inner-equiv 1) false)
							(list (quote get_column) outer-tv false outer-col false))))))
		acc))
		nil)))

(define qpl-limit-wrapper-correlation-filter (lambda (expr)
	(if (nil? expr) nil
		(list (quote if) expr true false))))

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
			(define sub (qpl-drop-static-aggregate-limit sub))
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
							(if (not (nil? uncorr-where))
								sub
								(begin
									/* Collect inner-equibound passthroughs needed by corr-where,
									plus equi-binding keys whose original outer predicate may
									have been absorbed by earlier cclass rewrites. The wrapper
									still needs those keys to constrain the per-domain LIMIT
									consumer instead of scanning every partition row. */
									(define passthrough-cols
										(merge_unique
											(qpl-collect-corr-passthroughs corr-where sub-inner-aliases)
											(qpl-limit-equibind-inner-cols
												outer-refs
												(qpp-tuple-condition sub)
												sub-inner-aliases)))
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
									/* The LIMIT rewrite synthesizes a derived table. Its
									uncorrelated WHERE can still contain inner_select
									markers that are local to this derived scope. Do
									not leak those markers to legacy untangle_query:
									immediately run the relational pipeline on the
									synthesized sub-tuple. */
									(define inner-sub-lowered
										(if (qpl-tuple-has-markers? inner-sub)
											(lower_to_scans_pass
												(unnest_pass_allow_free
													(lift_dep_joins_pass inner-sub)))
											inner-sub))
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
									(define equibind-corr
										(qpl-limit-equibind-wrapper-condition
											outer-refs
											(qpp-tuple-condition sub)
											sub-inner-aliases
											wrap-alias))
									(define wrapper-where
										(qpl-and-cond
											(qpl-and-cond rn-condition
												(qpl-limit-wrapper-correlation-filter retargeted-corr))
											(qpl-limit-wrapper-correlation-filter equibind-corr)))
									(qpp-rebuild-tuple
										schema
										(list (list wrap-alias schema inner-sub-lowered false nil))
										/* FLAT fields for wrapper too. */
										(list orig-field-name
											(list (quote get_column) wrap-alias false "__value" false))
										wrapper-where
										nil nil nil nil nil))))))))))))

/* qpl-collect-corr-passthroughs — find inner-side col refs in a correlated
WHERE expression, return the list of (tv col) pairs that need to be projected
through the window-bearing derived so the wrapper can reference them. */
(define qpl-collect-corr-passthroughs (lambda (expr inner-aliases)
	(if (nil? expr) '()
		(begin
			(define single-inner-alias (if (equal? (count inner-aliases) 1) (car inner-aliases) nil))
			(define all-refs (qpl-extract-col-refs-skip-nested expr))
			(define inner-refs (filter (map all-refs (lambda (rp) (match rp
				'(nil col) (if (nil? single-inner-alias) nil (list single-inner-alias col))
				'(tv col) (if (has? inner-aliases tv) rp nil)
				nil)))
				(lambda (rp) (not (nil? rp)))))
			/* Deduplicate */
			(reduce inner-refs (lambda (acc rp)
				(if (has? acc rp) acc (merge acc (list rp))))
				'())))))

/* qpl-retarget-refs — rewrite (get_column tv ti col ci) where tv ∈ from-aliases
to (get_column to-alias false col false). Used to retarget inner refs in the
correlated WHERE conjunct to use the wrapping derived's alias after the cols
get projected through as passthroughs. */
(define qpl-retarget-refs (lambda (expr from-aliases to-alias)
	(begin
		(define single-inner-alias (if (equal? (count from-aliases) 1) (car from-aliases) nil))
		(match expr
			'((symbol get_column) tv ti col ci)
			(if (or (has? from-aliases tv) (and (nil? tv) (not (nil? single-inner-alias))))
				(list (quote get_column) to-alias false col false)
				expr)
			'((quote get_column) tv ti col ci)
			(if (or (has? from-aliases tv) (and (nil? tv) (not (nil? single-inner-alias))))
				(list (quote get_column) to-alias false col false)
				expr)
			(cons head args)
			(if (list? args)
				(cons head (map args
					(lambda (a) (qpl-retarget-refs a from-aliases to-alias))))
				expr)
			_ expr))))

/* qpl-qualify-local-nil-refs — resolve unqualified refs that are provably
local to the current subquery. The unnest cclass pass only sees qualified
refs; leaving `owner` as tv=nil makes `owner = outer.id` invisible as an
inner/outer equivalence. We only qualify when exactly one local table exposes
the column, preserving SQL's outer-scope fallback for other nil refs. */
(define qpl-local-aliases-for-column (lambda (schemas col)
	(reduce_assoc schemas (lambda (acc alias cols)
		(if (reduce (coalesceNil cols '()) (lambda (found c)
			(or found (equal?? (c "Field") col))) false)
			(merge acc (list alias))
			acc))
		'())))

(define qpl-qualify-local-nil-refs-expr (lambda (expr schemas)
	(match expr
		'((symbol get_column) tv ti col ci)
		(if (nil? tv)
			(begin
				(define aliases (qpl-local-aliases-for-column schemas col))
				(if (equal? (count aliases) 1)
					(list (quote get_column) (car aliases) false col false)
					expr))
			expr)
		'((quote get_column) tv ti col ci)
		(if (nil? tv)
			(begin
				(define aliases (qpl-local-aliases-for-column schemas col))
				(if (equal? (count aliases) 1)
					(list (quote get_column) (car aliases) false col false)
					expr))
			expr)
		(cons head args)
		(if (or (not (list? args))
			(qpl-marker-head-eq? head "inner_select")
			(qpl-marker-head-eq? head "inner_select_in")
			(qpl-marker-head-eq? head "inner_select_exists"))
			expr
			(cons head (map args (lambda (a)
				(qpl-qualify-local-nil-refs-expr a schemas)))))
		_ expr)))

(define qpl-qualify-local-nil-refs-tuple (lambda (sub)
	(begin
		(define schemas (qpp-schemas-from-tables (qpp-tuple-tables sub)))
		(qpp-apply-to-tuple sub (lambda (expr)
			(qpl-qualify-local-nil-refs-expr expr schemas))))))

(define qpl-rewrite-correlated-limit-in-expr (lambda (expr)
	(match expr
		(cons sym args) (begin
			(define is-scalar (match sym
				(symbol inner_select) true
				(quote inner_select)  true
				'(quote inner_select) true
				'inner_select         true
				false))
			(if (and is-scalar (list? args) (equal? (count args) 1))
				(list sym (qpl-rewrite-correlated-limit-with-rownumber (nth args 0)))
				(if (list? args)
					(cons sym (map args
						(lambda (a) (qpl-rewrite-correlated-limit-in-expr a))))
					expr)))
		_ expr)))

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
		(if (list? args)
			(cons sym (map args
				(lambda (a) (qpl-qualify-walk a alias))))
			expr)
		_ expr)))

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

(define qpl-local-ref-info (lambda (expr inner-aliases)
	(begin
		(define info (qpl-col-ref-info expr))
		(define single-inner-alias (if (equal? (count inner-aliases) 1) (car inner-aliases) nil))
		(match info
			'(tv col)
			(if (nil? tv)
				(if (nil? single-inner-alias) nil (list single-inner-alias col))
				(if (has? inner-aliases tv) info nil))
			nil))))

(define qpl-outer-ref-info (lambda (expr inner-aliases)
	(match (qpl-col-ref-info expr)
		'(tv col)
		(if (and (not (nil? tv)) (not (has? inner-aliases tv)))
			(list tv col)
			nil)
		nil)))

(define qpl-local-ref-expr (lambda (info)
	(match info
		'(tv col) (list (quote get_column) tv false col false)
		nil)))

(define qpl-count-key-conjuncts (lambda (expr)
	(match expr
		'((symbol sql_truthy) inner) (qpl-count-key-conjuncts inner)
		'((quote sql_truthy) inner) (qpl-count-key-conjuncts inner)
		(cons head args)
		(if (or (equal? head (quote and)) (equal? head 'and))
			(reduce args (lambda (acc a)
				(merge acc (qpl-count-key-conjuncts a))) '())
			(list expr))
		(if (nil? expr) '() (list expr)))))

(define qpl-correlated-count-group-keys (lambda (condition inner-aliases)
	(reduce (qpl-count-key-conjuncts (coalesceNil condition true)) (lambda (acc part)
		(begin
			(define local-info (match part
				'((symbol equal??) lhs rhs)
				(if (and
					(not (nil? (qpl-local-ref-info lhs inner-aliases)))
					(not (nil? (qpl-outer-ref-info rhs inner-aliases))))
					(qpl-local-ref-info lhs inner-aliases)
					(if (and
						(not (nil? (qpl-local-ref-info rhs inner-aliases)))
						(not (nil? (qpl-outer-ref-info lhs inner-aliases))))
						(qpl-local-ref-info rhs inner-aliases)
						nil))
				'((quote equal??) lhs rhs)
				(if (and
					(not (nil? (qpl-local-ref-info lhs inner-aliases)))
					(not (nil? (qpl-outer-ref-info rhs inner-aliases))))
					(qpl-local-ref-info lhs inner-aliases)
					(if (and
						(not (nil? (qpl-local-ref-info rhs inner-aliases)))
						(not (nil? (qpl-outer-ref-info lhs inner-aliases))))
						(qpl-local-ref-info rhs inner-aliases)
						nil))
				'((symbol =) lhs rhs)
				(if (and
					(not (nil? (qpl-local-ref-info lhs inner-aliases)))
					(not (nil? (qpl-outer-ref-info rhs inner-aliases))))
					(qpl-local-ref-info lhs inner-aliases)
					(if (and
						(not (nil? (qpl-local-ref-info rhs inner-aliases)))
						(not (nil? (qpl-outer-ref-info lhs inner-aliases))))
						(qpl-local-ref-info rhs inner-aliases)
						nil))
				'((quote =) lhs rhs)
				(if (and
					(not (nil? (qpl-local-ref-info lhs inner-aliases)))
					(not (nil? (qpl-outer-ref-info rhs inner-aliases))))
					(qpl-local-ref-info lhs inner-aliases)
					(if (and
						(not (nil? (qpl-local-ref-info rhs inner-aliases)))
						(not (nil? (qpl-outer-ref-info lhs inner-aliases))))
						(qpl-local-ref-info rhs inner-aliases)
						nil))
				nil))
			(define key (qpl-local-ref-expr local-info))
			(if (or (nil? key) (has? acc key)) acc (merge acc (list key)))))
		'())))

(define qpl-make-count-subquery-for-exists (lambda (sub)
	(if (not (qpp-tuple? sub))
		(error "qpl-make-count-subquery-for-exists: sub is not a 7-tuple (likely UNION ALL — phase 5+)")
		(begin
			(define own-aliases (qpl-tuple-own-aliases sub))
			(define local-condition-group-keys
				(if (and
					(qpl-tuple-has-outer-refs? sub)
					(or (nil? (qpp-tuple-group sub)) (equal? (qpp-tuple-group sub) '())))
					(qpl-correlated-count-group-keys
						(qpp-tuple-condition sub)
						own-aliases)
					(qpp-tuple-group sub)))
			(qpp-rebuild-tuple
				(qpp-tuple-schema sub)
				(qpp-tuple-tables sub)
				(list "value" qpl-count-star-aggregate)
				(qpp-tuple-condition sub)
				local-condition-group-keys
				nil   /* HAVING dropped: the count above the group reduces to 0/n */
				'()   /* ORDER BY irrelevant for a scalar count */
				nil   /* LIMIT dropped */
				nil)))))

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
					(define qualified-sub-expr (qpl-qualify-outer-nil-refs raw-sub-expr sub-aliases))
					(define sub-expr (if (and
						(equal? (count sub-aliases) 1)
						(match qualified-sub-expr
							'((symbol get_column) alias_ _ _ _) (nil? alias_)
							'((quote get_column) alias_ _ _ _) (nil? alias_)
							false))
						(match qualified-sub-expr
							'((symbol get_column) _ ti col ci)
							(list (quote get_column) (car sub-aliases) ti col ci)
							'((quote get_column) _ ti col ci)
							(list (quote get_column) (car sub-aliases) ti col ci)
							qualified-sub-expr)
						qualified-sub-expr))
					(define count-condition
						(qpl-and-cond
							(list (quote equal??) a sub-expr)
							(qpp-tuple-condition sub)))
					(define local-condition-group-keys
						(if (and
							(qpl-tuple-has-outer-refs? sub)
							(or (nil? (qpp-tuple-group sub)) (equal? (qpp-tuple-group sub) '())))
							(qpl-correlated-count-group-keys
								count-condition
								sub-aliases)
							(qpp-tuple-group sub)))
					(qpp-rebuild-tuple
						(qpp-tuple-schema sub)
						(qpp-tuple-tables sub)
						(list "value" qpl-count-star-aggregate)
						count-condition
						local-condition-group-keys
						nil   /* HAVING dropped — see EXISTS comment */
						'() nil nil)))))))

(define qpl-union-first-field-name (lambda (branches) (begin
	(define first-branch (if (equal? (count branches) 0) nil (car branches)))
	(if (nil? first-branch)
		nil
		(begin
			(define first-fields (qpp-fields-to-pairs (qpp-tuple-fields first-branch)))
			(if (equal? (count first-fields) 1)
				(nth (nth first-fields 0) 0)
				nil))))))

(define qpl-make-count-subquery-for-union-in (lambda (a sub branches)
	(begin
		(define value-col (qpl-union-first-field-name branches))
		(if (nil? value-col)
			(error "lift_dep_joins_pass: UNION IN-subquery must project exactly one field")
			(begin
				(define union-alias (qpl-fresh-sq-alias))
				(define union-ref (list (quote get_column) union-alias false value-col false))
				(qpp-rebuild-tuple
					""
					(list (list union-alias "" sub false nil))
					(list "value" qpl-count-star-aggregate)
					(list (quote equal??) a union-ref)
					(list (list 1))
					nil
					'()
					nil
					nil))))))

(define qpl-make-count-subquery-for-union-exists (lambda (sub)
	(begin
		(define union-alias (qpl-fresh-sq-alias))
		(qpp-rebuild-tuple
			""
			(list (list union-alias "" sub false nil))
			(list "value" qpl-count-star-aggregate)
			true
			(list (list 1))
			nil
			'()
			nil
			nil))))

(define qpl-union-scalar-scan-lower-expr (lambda (expr union-alias)
	(match expr
		'((symbol get_column) alias_ _ col _) (if (or (nil? alias_) (equal?? alias_ union-alias))
			(symbol (concat union-alias "." col))
			(list (quote outer) (symbol (concat alias_ "." col))))
		'((quote get_column) alias_ _ col _) (if (or (nil? alias_) (equal?? alias_ union-alias))
			(symbol (concat union-alias "." col))
			(list (quote outer) (symbol (concat alias_ "." col))))
		(cons fsym fargs) (if (is_opaque_scope_sym fsym)
			expr
			(cons (qpl-union-scalar-scan-lower-expr fsym union-alias)
				(map fargs (lambda (a)
					(qpl-union-scalar-scan-lower-expr a union-alias)))))
		_ expr)))

(define qpl-union-in-scalar-scan-expr (lambda (target-expr sub branches) (begin
	(define value-col (qpl-union-first-field-name branches))
	(if (nil? value-col)
		(error "lift_dep_joins_pass: UNION IN-subquery must project exactly one field")
		(begin
			(define union-alias (qpl-fresh-sq-alias))
			(define filter-expr
				(list (quote equal??)
					(list (quote get_column) union-alias false value-col false)
					target-expr))
			(define scan-expr
				(list (quote scalar_scan)
					""
					sub
					(list (quote list) value-col)
					(list (quote lambda)
						(list (symbol (concat union-alias "." value-col)))
						(qpl-union-scalar-scan-lower-expr filter-expr union-alias))
					(list (quote list))
					(list (quote lambda) (list) true)
					(list (quote lambda) (list (quote acc) (quote item)) true)
					nil
					nil))
			(list (quote if)
				(list (quote nil?) target-expr)
				nil
				(list (quote not) (list (quote nil?) scan-expr))))))))

(define qpl-union-exists-scalar-scan-expr (lambda (sub) (begin
	(define scan-expr
		(list (quote scalar_scan)
			""
			sub
			(list (quote list))
			(list (quote lambda) (list) true)
			(list (quote list))
			(list (quote lambda) (list) true)
			(list (quote lambda) (list (quote acc) (quote item)) true)
			nil
			nil))
	(list (quote not) (list (quote nil?) scan-expr)))))

(define qpl-cols-for-alias (lambda (expr alias-name)
	(match expr
		'((symbol get_column) tv _ col _) (if (or (nil? tv) (equal?? tv alias-name))
			(list col)
			'())
		'((quote get_column) tv _ col _) (if (or (nil? tv) (equal?? tv alias-name))
			(list col)
			'())
		(cons sym args) (if (or (is_opaque_scope_sym sym) (not (list? args)))
			'()
			(reduce args (lambda (acc arg)
				(reduce (qpl-cols-for-alias arg alias-name) (lambda (acc2 col)
					(if (has? acc2 col) acc2 (merge acc2 (list col))))
					acc))
				'()))
		'())))

(define qpl-single-table-exists-scalar-scan-expr (lambda (sub) (match sub
	'(sq_schema sq_tables _sq_fields sq_condition sq_group sq_having sq_order sq_limit sq_offset)
	(if (and
		(equal? (count sq_tables) 1)
		(or (nil? sq_group) (equal? sq_group '()))
		(or (nil? sq_having) (equal? sq_having true))
		(or (nil? sq_order) (equal? sq_order '()))
		(or (nil? sq_limit) (> sq_limit 0))
		(or (nil? sq_offset) (equal? sq_offset 0)))
		(match (car sq_tables)
			'(tv tschema ttbl _isOuter tjoinexpr)
			(begin
				(define filter-expr
					(qpl-and-cond (coalesceNil tjoinexpr true) (coalesceNil sq_condition true)))
				(define filter-cols (qpl-cols-for-alias filter-expr tv))
				(define filter-params (map filter-cols (lambda (col)
					(symbol (concat tv "." col)))))
				(define scan-expr
					(list (quote scalar_scan)
						tschema
						ttbl
						(cons (quote list) filter-cols)
						(list (quote lambda)
							filter-params
							(qpl-union-scalar-scan-lower-expr filter-expr tv))
						(list (quote list))
						(list (quote lambda) (list) true)
						(list (quote lambda) (list (quote acc) (quote item)) true)
						nil
						nil))
				(list (quote not) (list (quote nil?) scan-expr)))
			_ nil)
		(if (equal? sq_limit 0) false nil))
	_ nil)))

(define qpl-single-table-in-scalar-scan-expr (lambda (target-expr sub) (match sub
	'(sq_schema sq_tables sq_fields sq_condition sq_group sq_having sq_order sq_limit sq_offset)
	(if (and
		(equal? (count sq_tables) 1)
		(or (nil? sq_group) (equal? sq_group '()))
		(or (nil? sq_having) (equal? sq_having true))
		(or (nil? sq_order) (equal? sq_order '()))
		(nil? sq_limit)
		(or (nil? sq_offset) (equal? sq_offset 0)))
		(match (car sq_tables)
			'(tv tschema ttbl _isOuter tjoinexpr)
			(begin
				(define first-field-expr-raw (match (qpp-fields-to-pairs sq_fields)
					(cons first-pair _) (nth first-pair 1)
					nil))
				(define first-field-expr (match first-field-expr-raw
					'((symbol get_column) alias_ ti col ci) (if (nil? alias_)
						(list (quote get_column) tv ti col ci)
						first-field-expr-raw)
					'((quote get_column) alias_ ti col ci) (if (nil? alias_)
						(list (quote get_column) tv ti col ci)
						first-field-expr-raw)
					first-field-expr-raw))
				(if (nil? first-field-expr)
					nil
					(begin
						(define filter-expr (qpl-and-cond
							(qpl-and-cond (coalesceNil tjoinexpr true) (coalesceNil sq_condition true))
							(list (quote equal??) first-field-expr target-expr)))
						(define filter-cols (qpl-cols-for-alias filter-expr tv))
						(define filter-params (map filter-cols (lambda (col)
							(symbol (concat tv "." col)))))
						(define scan-expr
							(list (quote scalar_scan)
								tschema
								ttbl
								(cons (quote list) filter-cols)
								(list (quote lambda)
									filter-params
									(qpl-union-scalar-scan-lower-expr filter-expr tv))
								(list (quote list))
								(list (quote lambda) (list) true)
								(list (quote lambda) (list (quote acc) (quote item)) true)
								nil
								nil))
						(list (quote if)
							(list (quote nil?) target-expr)
							nil
							(list (quote not) (list (quote nil?) scan-expr))))))
			_ nil)
		nil)
	_ nil)))

(define qpl-single-row-in-scalar-equality-expr (lambda (target-expr sub) (match sub
	'(sq_schema sq_tables sq_fields sq_condition sq_group sq_having sq_order sq_limit sq_offset)
	(if (and
		(equal? (count sq_tables) 1)
		(or (nil? sq_group) (equal? sq_group '()))
		(or (nil? sq_having) (equal? sq_having true))
		(or (nil? sq_order) (equal? sq_order '()))
		(nil? sq_limit)
		(or (nil? sq_offset) (equal? sq_offset 0)))
		(match (car sq_tables)
			'(_tv _tschema ttbl _isOuter _tjoinexpr)
			(match ttbl
				'(_ _ _ _ _ _ _ inner_limit inner_offset)
				(if (and
					(not (nil? inner_limit))
					(<= inner_limit 1)
					(or (nil? inner_offset) (equal? inner_offset 0)))
					(list (quote if)
						(list (quote nil?) target-expr)
						nil
						(list (quote equal??) target-expr (list (quote inner_select) sub)))
					nil)
				_ nil)
			_ nil)
		nil)
	_ nil)))

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
(define qpl-union-all-raw-parts (lambda (sub) (match sub
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

(define qpl-union-branch-simple-membership? (lambda (branch)
	(if (not (qpp-tuple? branch))
		false
		(and
			(or (nil? (qpp-tuple-group branch)) (equal? (qpp-tuple-group branch) '()))
			(or (nil? (qpp-tuple-having branch)) (equal? (qpp-tuple-having branch) true))
			(or (nil? (qpp-tuple-order branch)) (equal? (qpp-tuple-order branch) '()))
			(nil? (qpp-tuple-limit branch))
			(or (nil? (qpp-tuple-offset branch)) (equal? (qpp-tuple-offset branch) 0))
			(reduce (coalesceNil (qpp-tuple-tables branch) '()) (lambda (ok td) (and ok (match td
				'(_ tschema ttbl _ _)
				(and
					(string? ttbl)
					(not (nil? (try (lambda () (get_schema tschema ttbl)) (lambda (e) nil)))))
				false)))
				true)))))

(define qpl-union-all-parts (lambda (sub) (begin
	(define branches (qpl-union-all-raw-parts sub))
	(if (and
		(not (nil? branches))
		(reduce branches (lambda (ok branch)
			(and ok (qpl-union-branch-simple-membership? branch)))
			true))
		branches
		nil))))

/* qpl-and-or-from-branches — build `(or BR1 BR2 ...)` for IN-UNION rewrite
(positive: any branch matches) or `(and BR1 BR2 ...)` for NOT-IN rewrite
(negative: every branch fails). Empty branches: false for or, true for and. */
(define qpl-or-from-list (lambda (terms)
	(if (equal? (count terms) 0) false
		(if (equal? (count terms) 1) (nth terms 0)
			(reduce (cdr terms) (lambda (acc t)
				(list (quote or) acc t)) (car terms))))))

(define qpl-and-from-list (lambda (terms)
	(reduce terms (lambda (acc term)
		(qpl-and-cond acc term))
		true)))

(define qpl-equality-other-side (lambda (term expr) (match term
	(cons eq-sym (cons lhs (cons rhs '())))
	(if (or
		(equal?? eq-sym (quote equal??))
		(equal?? eq-sym (symbol equal??))
		(equal?? eq-sym (quote equal?))
		(equal?? eq-sym (symbol equal?)))
		(if (equal? (serialize lhs) (serialize expr))
			rhs
			(if (equal? (serialize rhs) (serialize expr))
				lhs
				nil))
		nil)
	_ nil)))

(define qpl-exists-union-branch-in-marker (lambda (branch outer-aliases) (begin
	(if (not (qpp-tuple? branch))
		nil
		(begin
			(define branch-fields (qpp-fields-to-pairs (qpp-tuple-fields branch)))
			(if (not (equal? (count branch-fields) 1))
				nil
				(begin
					(define projected-expr (nth (nth branch-fields 0) 1))
					(define cond-terms (qpl-and-conjuncts (qpp-tuple-condition branch)))
					(define eq-info (reduce cond-terms (lambda (found term)
						(if (not (nil? found))
							found
							(begin
								(define target-expr (qpl-equality-other-side term projected-expr))
								(if (nil? target-expr)
									nil
									(list term target-expr)))))
						nil))
					(match eq-info
						'(matched-term target-expr)
						(begin
							(define branch-without-eq
								(qpp-rebuild-tuple
									(qpp-tuple-schema branch)
									(qpp-tuple-tables branch)
									(qpp-tuple-fields branch)
									(qpl-and-from-list (filter cond-terms (lambda (term)
										(not (equal? term matched-term)))))
									(qpp-tuple-group branch)
									(qpp-tuple-having branch)
									(qpp-tuple-order branch)
									(qpp-tuple-limit branch)
									(qpp-tuple-offset branch)))
							(list
								(quote inner_select_in)
								(qpl-qualify-outer-nil-refs target-expr outer-aliases)
								branch-without-eq))
						_ nil))))))))

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
			(define raw-branches (qpl-union-all-raw-parts sub))
			(define branches (qpl-union-all-parts sub))
			(if (not (nil? branches))
				/* IN (UNION ALL of branches) → (lhs IN br1) OR (lhs IN br2) OR …
				Re-run rewrite on each new marker to recurse properly. */
				(qpl-rewrite-in-exists
					(qpl-or-from-list (map branches (lambda (br)
						(list (quote inner_select_in) (qpl-marker-lhs expr) br))))
					outer-aliases)
				(if (not (nil? raw-branches))
					(qpl-rewrite-in-exists
						(qpl-or-from-list (map raw-branches (lambda (br)
							(list (quote inner_select_in) (qpl-marker-lhs expr) br))))
						outer-aliases)
					(coalesce
						(if (not (qpl-tuple-has-outer-refs? sub))
							(qpl-single-table-in-scalar-scan-expr
								(qpl-qualify-outer-nil-refs
									(qpl-rewrite-in-exists (qpl-marker-lhs expr) outer-aliases)
									outer-aliases)
								sub)
							nil)
						(qpl-single-row-in-scalar-equality-expr
							(qpl-qualify-outer-nil-refs
								(qpl-rewrite-in-exists (qpl-marker-lhs expr) outer-aliases)
								outer-aliases)
							sub)
						(qpl-wrap-as-count-gt-zero
							(qpl-make-count-subquery-for-in
								(qpl-qualify-outer-nil-refs
									(qpl-rewrite-in-exists (qpl-marker-lhs expr) outer-aliases)
									outer-aliases)
								sub))))))
		(if (equal? k (quote inner_select_exists))
			(begin
				(define sub (qpl-marker-subquery expr))
				(define raw-branches (qpl-union-all-raw-parts sub))
				(define branches (qpl-union-all-parts sub))
				(if (not (nil? branches))
					(begin
						(define branch-in-markers (filter
							(map branches (lambda (br)
								(qpl-exists-union-branch-in-marker br outer-aliases)))
							(lambda (marker) (not (nil? marker)))))
						(if (equal? (count branch-in-markers) (count branches))
							(qpl-rewrite-in-exists
								(qpl-or-from-list branch-in-markers)
								outer-aliases)
							/* Fallback for branch shapes where EXISTS cannot be expressed
							as a value-membership probe yet. */
							(qpl-rewrite-in-exists
								(qpl-or-from-list (map branches (lambda (br)
									(list (quote inner_select_exists) br))))
								outer-aliases)))
					(if (not (nil? raw-branches))
						(qpl-rewrite-in-exists
							(qpl-or-from-list (map raw-branches (lambda (br)
								(list (quote inner_select_exists) br))))
							outer-aliases)
						(coalesce
							(if (not (qpl-tuple-has-outer-refs? sub))
								(qpl-single-table-exists-scalar-scan-expr sub)
								nil)
							(qpl-wrap-as-count-gt-zero
								(qpl-make-count-subquery-for-exists sub))))))
			(match expr
				(cons sym args) (if (list? args)
					(cons sym (map args
						(lambda (a) (qpl-rewrite-in-exists a outer-aliases))))
					expr)
				_ expr))))))

/* qpl-rewrite-in-exists-fields — apply the rewrite to each projection. */
(define qpl-rewrite-in-exists-fields (lambda (fields outer-aliases)
	(map (qpp-fields-to-pairs fields) (lambda (pair) (match pair
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
	(define rewritten-tables (map (coalesceNil (qpp-tuple-tables t) '()) (lambda (td) (match td
		'(alias schema tname isOuter joinExpr)
		(list alias schema
			(if (qpp-tuple? tname)
				(qpl-rewrite-in-exists-tuple tname)
				tname)
			isOuter
			(qpl-rewrite-in-exists joinExpr outer-aliases))
		td))))
	(qpp-rebuild-tuple
		(qpp-tuple-schema t)
		rewritten-tables
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
	(concat "nq_" (string (qpl-sq-counter "n"))))))

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
		(if (and (list? inner) (equal? (count inner) 4)
			(qpl-marker-head-eq? (nth inner 0) "if")
			(equal? (nth inner 2) 0)
			(equal? (nth inner 3) 1))
			(begin
				(define nil-check (nth inner 1))
				(and (list? nil-check)
					(equal? (count nil-check) 2)
					(qpl-marker-head-eq? (nth nil-check 0) "nil?")))
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
			(if (not (qpp-tuple? sub))
				(error (concat "qpl-substitute-markers: malformed inner_select marker "
					(serialize expr) " sub=" (serialize sub)))
				nil)
			/* Deduplicate marker-free scalar subqueries within the current lift
			pass. Correlated helpers are safe to share only in this local acc scope:
			they see the same outer row-domain and lower to the same dep-join
			boundary. Nested-marker subqueries still get their own alias because
			their inner helpers need their own top-down context. */
			(define can-reuse
				(not (qpl-tuple-has-markers? sub)))
			(define cache-key (if can-reuse
				(concat
					(if (qpl-tuple-has-outer-refs? sub) "corr:" "uncorr:")
					(fnv_hash (serialize (normalize_canonical_aliases sub))))
				nil))
			(define cached-alias (if can-reuse
				(get_assoc (coalesceNil (acc "uncorr-cache") '()) cache-key)
				nil))
			(define sq-alias (if (nil? cached-alias)
				(qpl-fresh-sq-alias)
				cached-alias))
			(if (nil? cached-alias)
				(begin
					(acc "list" (merge (coalesceNil (acc "list") '())
						(list (list sq-alias sub))))
					(if can-reuse
						(acc "uncorr-cache"
							(set_assoc (coalesceNil (acc "uncorr-cache") '())
								cache-key sq-alias))
						nil))
				nil)
			(qpl-wrap-with-aggregate-neutral (qpl-sq-col sq-alias) sub))
		(if (not (nil? k))
			(error (concat "lift_dep_joins_pass: marker kind " (string k)
				" not yet supported (Phase 3+). Only scalar inner_select is handled."))
			(match expr
				(cons sym args) (if (list? args)
					(cons sym (map args
						(lambda (a) (qpl-substitute-markers a acc))))
					expr)
				_ expr))))))

/* qpl-substitute-fields — apply qpl-substitute-markers to every projection
expression in a fields list, accumulating subqueries into acc. */
(define qpl-substitute-fields (lambda (fields acc)
	(reduce (qpp-fields-to-pairs fields) (lambda (out pair) (match pair
		'(name expr)
		(merge out (list (list name (qpl-substitute-markers expr acc))))
		(merge out (list pair))))
		'())))

/* qpl-simplify-where-in-null-branch — parser-lowered IN uses SQL three-valued
logic: (if lhs-is-null nil (if match true (if rhs-has-null nil false))).
Inside WHERE, nil/UNKNOWN and false both reject the row, so the RHS NULL branch
is equivalent to false and the predicate reduces to the match test. This keeps
decorrelation to one COUNT helper instead of materializing an unused null-count
helper for positive IN predicates. */
(define qpl-if-head? (lambda (head) (match head
	(symbol if) true
	(quote if) true
	'(quote if) true
	'if true
	false)))

(define qpl-simplify-where-in-null-branch (lambda (expr)
	(match expr
		(cons head args)
		(if (and (qpl-if-head? head) (list? args) (equal? (count args) 3))
			(begin
				(define lhs-null-then (nth args 1))
				(define match-branch (nth args 2))
				(match match-branch
					(cons match-head match-args)
					(if (and (qpl-if-head? match-head) (list? match-args)
						(equal? (count match-args) 3))
						(begin
							(define match-expr (nth match-args 0))
							(define match-then (nth match-args 1))
							(define rhs-null-branch (nth match-args 2))
							(match rhs-null-branch
								(cons rhs-head rhs-args)
								(if (and (qpl-if-head? rhs-head) (list? rhs-args)
									(equal? (count rhs-args) 3)
									(nil? lhs-null-then)
									(equal? match-then true)
									(nil? (nth rhs-args 1))
									(equal? (nth rhs-args 2) false))
									(qpl-simplify-where-in-null-branch match-expr)
									(cons head (map args qpl-simplify-where-in-null-branch)))
								(cons head (map args qpl-simplify-where-in-null-branch))))
						(cons head (map args qpl-simplify-where-in-null-branch)))
					(cons head (map args qpl-simplify-where-in-null-branch))))
			(if (list? args)
				(cons head (map args qpl-simplify-where-in-null-branch))
				expr))
		_ expr)))

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
	(not (equal? (qpp-fields-to-pairs orig) (qpp-fields-to-pairs sub)))))

/* qpl-strip-marker-dependent-fields — build the projection list for the
outer leaf before dep-joins introduce sq_N aliases. Projection expressions
that contained subquery markers are replaced by NULL placeholders; the real
expressions are applied by a qpir-map above the dep-join chain where sq_N is
in scope. */
(define qpl-strip-marker-dependent-fields (lambda (fields)
	(map (qpp-fields-to-pairs fields) (lambda (pair) (match pair
		'(name expr) (if (> (count (qpl-collect-markers expr)) 0)
			(list name nil)
			pair)
		pair)))))

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
	(reduce (qpp-fields-to-pairs fields) (lambda (acc pair) (match pair
		'(name expr) (if (qpl-is-aggregate-expr? expr)
			(merge acc (list (list name expr)))
			acc)
		acc)) '())))

/* qpl-collect-non-aggregate-fields — return the list of non-aggregate fields
in the same order they appear. Errors loudly if a field contains an aggregate
nested inside a non-bare expression (e.g. AVG = (SUM/COUNT)) — phase 5 will
handle those. */
(define qpl-collect-non-aggregate-fields (lambda (fields)
	(reduce (qpp-fields-to-pairs fields) (lambda (acc pair) (match pair
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

/* qpl-first-row-identity-expr — pick a stable row identity from the first real
table in a tuple, preferring PRIMARY KEY. Used when WHERE-subquery lifting
moves GROUP BY above a decorrelated join: COUNT(*) must count original rows,
not the deduplicated prejoin domain. */
(define qpl-first-row-identity-expr (lambda (tables)
	(reduce (coalesceNil tables '()) (lambda (found td) (match td
		'(alias tschema ttbl _ _)
		(if (not (nil? found)) found
			(if (qpp-tuple? ttbl) nil
				(begin
					(define resolved-alias (if (nil? alias) ttbl alias))
					(define cols (try (lambda () (get_schema tschema ttbl))
						(lambda (e) '())))
					(define pk (reduce cols (lambda (acc coldef)
						(if (not (nil? acc)) acc
							(if (equal? (coldef "Key") "PRI") (coldef "Field") nil)))
						nil))
					(define fallback (match cols
						(cons first _) (first "Field")
						nil))
					(define col (coalesce pk fallback))
					(if (nil? col) nil
						(list (quote get_column) resolved-alias false col false)))))
		found)) nil)))

(define qpl-rewrite-count-star-aggs-with-rowid (lambda (agg-fields rowid-expr)
	(if (nil? rowid-expr) agg-fields
		(map agg-fields (lambda (pair) (match pair
			'(name (cons head rest))
			(begin
				(define agg-inner (nth rest 0))
				(define agg-reducer (nth rest 1))
				(define agg-init (nth rest 2))
				(if (equal? agg-inner 1)
					(list name (list head
						(list (quote if) (list (quote nil?) rowid-expr) 0 1)
						agg-reducer agg-init))
					pair))
			pair))))))

/* qpl-leaf-input-fields-for-aggs — produce the projection list the underlying
qpir-leaf must expose so the qpir-groupby above can compute its aggregates.

For an aggregate (aggregate inner reducer init):
- The leaf must retain the `inner` expression's dependencies.
- The qpir-groupby's aggregate uses the real `inner` expression directly.

This avoids resolving the aggregate input through the output alias. In
materialized grouped derived tables, that alias belongs to the grouped output
and may otherwise become a nil-valued group helper instead of the row-local
source expression.

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
			(define agg-after (list name
				(list (quote aggregate)
					agg-inner
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
(define qpl-dropped-limit-marker-field? (lambda (pair)
	(match pair
		'("__qpl_dropped_limit" _) true
		_ false)))

(define qpl-visible-scalar-fields (lambda (fields)
	(filter (qpp-fields-to-pairs fields)
		(lambda (pair) (not (qpl-dropped-limit-marker-field? pair))))))

(define qpl-internal-marker-fields (lambda (fields)
	(filter (qpp-fields-to-pairs fields) qpl-dropped-limit-marker-field?)))

(define qpl-rename-first-field-to-value (lambda (sub) (begin
	/* Sub's fields can be EITHER flat (name1 expr1 …) (parser shape) or
	list-of-pairs ((name1 expr1) …) (pipeline-internal shape). Normalize
	to pairs via qpp-fields-to-pairs before counting/extracting. */
	(define fields-as-pairs (qpl-visible-scalar-fields (qpp-tuple-fields sub)))
	(define marker-fields (qpl-internal-marker-fields (qpp-tuple-fields sub)))
	(if (not (equal? (count fields-as-pairs) 1))
		(error (concat "qpl-rename-first-field-to-value: expected 1 field, found "
			(string (count fields-as-pairs)))) nil)
	(define field-pair (nth fields-as-pairs 0))
	(define field-expr (nth field-pair 1))
	(qpp-rebuild-tuple
		(qpp-tuple-schema sub)
		(qpp-tuple-tables sub)
		(merge
			(list (list "value" field-expr))
			marker-fields)
		(qpp-tuple-condition sub)
		(qpp-tuple-group sub)
		(qpp-tuple-having sub)
		(qpp-tuple-order sub)
		(qpp-tuple-limit sub)
		(qpp-tuple-offset sub)))))

(define qpl-wrap-inner-subquery (lambda (sub)
	(if (not (qpp-tuple? sub))
		(error (concat "qpl-wrap-inner-subquery: sub is not a 7-tuple: " (serialize sub)))
		(begin
			(define sub (qpl-qualify-local-nil-refs-tuple sub))
			/* Normalize to pairs — sub may be parser-flat or pipeline-pairs. */
			(define fields (qpl-visible-scalar-fields (qpp-tuple-fields sub)))
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
					(define can-decompose-uncorrelated-count
						(and
							(not (nil? (qpl-sub-aggregate-neutral leaf-tuple)))
							(equal? (count (coalesceNil (qpp-tuple-group leaf-tuple) '())) 0)
							(or (nil? (qpp-tuple-having leaf-tuple)) (equal? (qpp-tuple-having leaf-tuple) true))
							(equal? (count (coalesceNil (qpp-tuple-order leaf-tuple) '())) 0)
							(nil? (qpp-tuple-limit leaf-tuple))
							(nil? (qpp-tuple-offset leaf-tuple))))
					(if (qpl-tuple-has-outer-refs? leaf-tuple)
						/* Correlated: existing decompose-or-hoist logic */
						(if (qpl-needs-decompose? leaf-tuple)
							(qpl-build-groupby-wrapped-inner leaf-tuple)
							(qpl-build-simple-leaf-inner leaf-tuple))
						/* Uncorrelated: plain scalar subqueries pass through as-is.
						COUNT-like aggregates are decomposed too, because their empty
						input must still provide the neutral 0 via the scalar
						COALESCE contract; a materialized empty leaf would otherwise
						filter the outer row in legacy lowering. */
						(if can-decompose-uncorrelated-count
							(qpl-build-groupby-wrapped-inner leaf-tuple)
							(qpir-leaf leaf-tuple))))
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
			(cons head args) (if (list? args)
				(cons head (map args
					(lambda (a) (qpl-extract-aggregates a agg-acc))))
				expr)
			_ expr))))

(define qpl-build-groupby-wrapped-inner (lambda (sub) (begin
	(define fields-pairs (qpl-visible-scalar-fields (qpp-tuple-fields sub)))
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

(define qpl-table-joinexpr-marker-count (lambda (td)
	(if (or (nil? td) (not (list? td)) (< (count td) 5))
		0
		(count (qpl-collect-markers (nth td 4))))))

(define qpl-tuple-joinexpr-has-markers? (lambda (t)
	(> (reduce (coalesceNil (qpp-tuple-tables t) '()) (lambda (acc td)
		(+ acc (qpl-table-joinexpr-marker-count td))) 0) 0)))

(define qpl-empty-chain-leaf (lambda (schema tables)
	(qpir-leaf (qpp-rebuild-tuple
		schema
		tables
		'()
		true
		nil
		nil
		nil
		nil
		nil))))

(define qpl-chain-dep-markers (lambda (left markers)
	(reduce markers (lambda (left-acc pair) (match pair
		'(sq-alias sub)
		(qpir-dep-join true left-acc (qpl-wrap-inner-subquery sub) '() sq-alias)
		left-acc))
		left)))

(define qpl-append-table-to-chain (lambda (schema left td)
	(if (nil? left)
		(qpl-empty-chain-leaf schema (list td))
		(qpir-join (quote inner) true left
			(qpl-empty-chain-leaf schema (list td)) nil))))

(define qpl-build-table-chain-with-join-markers (lambda (t acc)
	(begin
		(define schema (qpp-tuple-schema t))
		(reduce (coalesceNil (qpp-tuple-tables t) '()) (lambda (left td) (match td
			'(alias tschema ttbl isOuter joinexpr)
			(begin
				(define before-list (coalesceNil (acc "list") '()))
				(define sub-joinexpr
					(if (nil? joinexpr) nil
						(qpl-substitute-markers joinexpr acc)))
				(define after-list (acc "list"))
				(define new-markers
					(filter after-list (lambda (entry)
						(not (has? before-list entry)))))
				(define left-with-deps
					(if (equal? (count new-markers) 0)
						left
						(if (nil? left)
							(error "lift_dep_joins_pass: JOIN ON marker on first table has no left domain")
							(qpl-chain-dep-markers left new-markers))))
				(qpl-append-table-to-chain schema left-with-deps
					(list alias tschema ttbl isOuter sub-joinexpr)))
			(qpl-append-table-to-chain schema left td)))
			nil))))

/* Derived-table boundaries are projection/rename scopes, not scalar-subquery
fallback zones. If a FROM-subquery still contains lifted markers after the
tuple-local IN/EXISTS rewrite, lower that subquery through the same Neumann
pipeline before the outer tuple is assembled. The result is still a 7-tuple
source for the outer flattening pass; no full-table materialization is added. */
(define qpl-lower-derived-table-markers (lambda (t)
	(if (not (qpp-tuple? t))
		t
		(begin
			(define rewritten-tables (map (coalesceNil (qpp-tuple-tables t) '()) (lambda (td) (match td
				'(alias schema tname isOuter joinExpr)
				(begin
					(define lowered-tname
						(if (qpp-tuple? tname)
							(begin
								(define nested (qpl-lower-derived-table-markers tname))
								(if (qpl-tuple-has-markers? nested)
									(lower_to_scans_pass
										(unnest_pass_allow_free
											(lift_dep_joins_pass nested)))
									nested))
							tname))
					(list alias schema lowered-tname isOuter joinExpr))
				td))))
			(qpp-rebuild-tuple
				(qpp-tuple-schema t)
				rewritten-tables
				(qpp-tuple-fields t)
				(qpp-tuple-condition t)
				(qpp-tuple-group t)
				(qpp-tuple-having t)
				(qpp-tuple-order t)
				(qpp-tuple-limit t)
				(qpp-tuple-offset t))))))

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
			(define t-derived (qpl-lower-derived-table-markers t-prime))
			(qpl-drop-redundant-qpir-limits
				(if (not (qpl-tuple-has-markers? t-derived))
					(qpir-leaf t-derived)
					(qpl-lift-with-markers t-derived)))))))

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

	/* JOIN ON markers must be introduced before the table whose ON-clause
	reads them. Treat them separately from projection/WHERE/HAVING markers,
	which can be applied above the finished table chain. */
	(define has-joinexpr-markers (qpl-tuple-joinexpr-has-markers? t))
	(define join-acc (newsession))
	(join-acc "list" '())
	(define join-chain
		(if has-joinexpr-markers
			(qpl-build-table-chain-with-join-markers t join-acc)
			nil))

	/* Collect + substitute all scalar markers from fields, condition, and
	HAVING. Order matters: fields → condition → having for deterministic
	sq_N numbering. */
	(define acc (newsession))
	(acc "list" '())
	(define orig-fields (qpp-tuple-fields t))
	(define sub-fields (qpl-substitute-fields orig-fields acc))
	(define orig-cond (qpl-simplify-where-in-null-branch (qpp-tuple-condition t)))
	(define sub-cond (qpl-substitute-markers orig-cond acc))
	(define orig-having (qpp-tuple-having t))
	(define sub-having
		(if (nil? orig-having) nil
			(qpl-substitute-markers orig-having acc)))
	(define having-has-markers
		(if (nil? orig-having) false
			(> (count (qpl-collect-markers orig-having)) 0)))
	(define markers (acc "list"))

	(if (and (not has-joinexpr-markers) (equal? (count markers) 0))
		/* Nothing actually got lifted (defensive: should not happen because
		qpl-tuple-has-markers? said yes). Fall back to leaf wrap. */
		(qpir-leaf t)
		(begin
			/* Build outer leaf: local fields only, WHERE replaced by true
			(real condition is re-applied above the dep-join chain so its
			sq.value references can resolve to the dep-join's right side).
			HAVING follows the same rule when it contains markers.
			Projection expressions that reference sq.value are applied as a
			qpir-map after the chain; the leaf cannot see sq aliases yet. */
			(define cond-has-markers (> (count (qpl-collect-markers orig-cond)) 0))
			(define group-keys (coalesceNil (qpp-tuple-group t) '()))
			(define operator-group-needed
				(and (> (count group-keys) 0)
					(or cond-has-markers has-joinexpr-markers)))
			(define agg-fields (if operator-group-needed
				(qpl-collect-aggregates-in-fields orig-fields) '()))
			(define count-star-agg-field? (lambda (pair) (match pair
				'(_name (cons _head rest))
				(equal? (nth rest 0) 1)
				false)))
			(define direct-agg-fields (if operator-group-needed
				(filter agg-fields count-star-agg-field?)
				'()))
			(define decomposed-agg-fields (if operator-group-needed
				(filter agg-fields (lambda (pair) (not (count-star-agg-field? pair))))
				'()))
			(define rewritten-direct-agg-fields (if operator-group-needed
				(if cond-has-markers
					(qpl-rewrite-count-star-aggs-with-rowid direct-agg-fields
						(qpl-first-row-identity-expr (qpp-tuple-tables t)))
					direct-agg-fields)
				'()))
			(define agg-projection-parts (if operator-group-needed
				(qpl-leaf-and-agg-projections decomposed-agg-fields)
				(list '() '())))
			(define non-agg-fields (if operator-group-needed
				(qpl-collect-non-aggregate-fields orig-fields) '()))
			(define sub-non-agg-fields (if operator-group-needed
				(qpl-collect-non-aggregate-fields sub-fields) '()))
			(define leaf-fields (if operator-group-needed
				(merge non-agg-fields (nth agg-projection-parts 0))
				(qpl-strip-marker-dependent-fields orig-fields)))
			(define field-is-group-key? (lambda (expr)
				(reduce group-keys (lambda (found key-expr)
					(or found (equal? expr key-expr)))
					false)))
			(define group-value-aggs (if operator-group-needed
				(filter
					(map sub-non-agg-fields (lambda (pair) (match pair
						'(name expr)
						(if (field-is-group-key? expr)
							nil
							(list name (list (quote aggregate) expr (quote group_any_value_reduce) nil)))
						nil)))
					(lambda (pair) (not (nil? pair))))
				'()))
			(define group-aggs (if operator-group-needed
				(merge
					rewritten-direct-agg-fields
					(nth agg-projection-parts 1)
					group-value-aggs)
				'()))
			(define leaf-having (if having-has-markers nil orig-having))
			(define outer-leaf
				(if has-joinexpr-markers
					join-chain
					(qpir-leaf (qpp-rebuild-tuple
						(qpp-tuple-schema t)
						(qpp-tuple-tables t)
						leaf-fields
						true
						(if operator-group-needed nil (qpp-tuple-group t))
						(if operator-group-needed nil leaf-having)
						(qpp-tuple-order t)
						(qpp-tuple-limit t)
						(qpp-tuple-offset t)))))

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
				(if (or (nil? sub-cond)
					(equal? sub-cond true)
					(equal? sub-cond (quote true)))
					chained
					(qpir-select sub-cond chained)))
			(define after-group
				(if operator-group-needed
					(qpir-groupby group-keys group-aggs leaf-having after-where)
					after-where))
			(define after-having
				(if having-has-markers
					(qpir-select sub-having after-group)
					after-group))

			(if (or has-joinexpr-markers (qpl-fields-touched? orig-fields sub-fields))
				(qpir-map sub-fields after-having)
				after-having))))))
