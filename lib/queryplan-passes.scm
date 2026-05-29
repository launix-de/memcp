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
== Layer 2 — Pre-passes (7-tuple → 7-tuple normalizations) ==

Per the neumann_compiler_plan, the L2 pipeline normalizes the parser-emitted
7-tuple before lift_dep_joins_pass (Day 3) lifts it into the L1 operator IR.

A 7-tuple has the parser shape:
  (schema tables fields condition group having order limit offset)

Each pre-pass is a PURE function — no shared mutable state, no sq_cache
mutation, no side effects. Inputs in, outputs out. This is the contract
that distinguishes the L2 pipeline from the legacy untangle_query.

Passes implemented in this module:
  1. [alias_normalize_pass tuple] →
       7-tuple with (get_column alias ...) rewritten so alias is the
       visible occurrence (drops NUL-separator provenance from derived-table
       flattening, picks the visible side of (visible_alias canonical) pairs).
  2. [column_resolve_pass tuple schemas] →
       7-tuple with (get_column alias ti col ci) resolved against the
       supplied schemas. After this pass ti/ci are no longer needed; column
       references use canonical casing.

Both passes delegate the per-expression rewrite to existing helpers from
lib/queryplan.scm — the new code is purely the 7-tuple-shaped wrapper.
Once the legacy untangle_query is removed, the helpers move into this module.

[derived_table_inline_pass] is intentionally NOT in this module yet — its
implementation (FAQ §36) is substantial enough to warrant its own commit;
it will land before lift_dep_joins_pass needs it.
*/

/* ==================== 7-tuple shape helpers ==================== */

/* qpp-tuple? — true when value matches the 9-element 7-tuple shape. */
(define qpp-tuple? (lambda (t)
	(and (not (nil? t)) (list? t) (equal? (count t) 9))))

(define qpp-tuple-schema    (lambda (t) (nth t 0)))
(define qpp-tuple-tables    (lambda (t) (nth t 1)))
(define qpp-tuple-fields    (lambda (t) (nth t 2)))
(define qpp-tuple-condition (lambda (t) (nth t 3)))
(define qpp-tuple-group     (lambda (t) (nth t 4)))
(define qpp-tuple-having    (lambda (t) (nth t 5)))
(define qpp-tuple-order     (lambda (t) (nth t 6)))
(define qpp-tuple-limit     (lambda (t) (nth t 7)))
(define qpp-tuple-offset    (lambda (t) (nth t 8)))

/* qpp-rebuild-tuple — rebuild a 7-tuple from its parts (used by passes). */
(define qpp-rebuild-tuple (lambda (schema tables fields condition group having order limit offset)
	(list schema tables fields condition group having order limit offset)))

/* qpp-fields-to-pairs — convert FLAT parser fields (name1 expr1 name2 expr2 …)
into list-of-pairs ((name1 expr1) (name2 expr2) …). The flat form is what
sql-parser.scm produces via (merge cols); my pipeline operates on pairs for
clarity. Idempotent on already-pairs input. */
(define qpp-fields-to-pairs (lambda (fields) (begin
	(define fs (coalesceNil fields (list)))
	(if (equal? (count fs) 0) (list)
		/* Detect if already pairs: first element is a 2-element list. */
		(if (and (list? (nth fs 0)) (equal? (count (nth fs 0)) 2))
			fs
			/* Flat: pair up via stride 2 */
			(map (produceN (/ (count fs) 2)) (lambda (i)
				(list (nth fs (* i 2)) (nth fs (+ (* i 2) 1))))))))))

/* qpp-fields-to-flat — convert list-of-pairs back to flat (kN vN kN+1 vN+1 …).
This is the inverse — used at the boundary between my pipeline (pairs) and
legacy consumers (flat). */
(define qpp-fields-to-flat (lambda (fields)
	(reduce (coalesceNil fields (list)) (lambda (acc pair) (match pair
		'(name expr) (merge acc (list name expr))
		(merge acc (list pair))))
		(list))))

/* qpp-map-fields — apply fn to every projection expression in a fields list.
Accepts both flat and pair forms; converts to pairs internally, then back to
pairs (caller decides flattening at the boundary). */
(define qpp-map-fields (lambda (fields fn)
	(map (qpp-fields-to-pairs fields) (lambda (pair) (match pair
		'(name expr) (list name (fn expr))
		pair)))))

/* qpp-map-order — apply fn to every order expression. order: ((expr dir) ...).
Preserves nil as nil (legacy code distinguishes nil from empty-list ()). */
(define qpp-map-order (lambda (order fn)
	(if (nil? order) nil
		(map order (lambda (item) (match item
			'(expr dir) (list (fn expr) dir)
			item))))))

/* qpp-map-group — apply fn to every group-by expression. group: (expr ...).
Preserves nil as nil (legacy treats nil-group = "no GROUP BY" vs empty-list
= "explicit empty group" differently in the count_distinct 2-stage path). */
(define qpp-map-group (lambda (group fn)
	(if (nil? group) nil
		(map group fn))))

/* qpp-apply-to-tuple — apply expression-rewrite fn to every expression-bearing
slot of a 7-tuple (fields, condition, group, having, order). limit/offset are
scalar values, not expressions, so they pass through. schema and tables also
pass through — alias/schema lookup is not part of expression rewriting. */
(define qpp-apply-to-tuple (lambda (t fn) (qpp-rebuild-tuple
	(qpp-tuple-schema t)
	(qpp-tuple-tables t)
	(qpp-map-fields (qpp-tuple-fields t) fn)
	(fn (qpp-tuple-condition t))
	(qpp-map-group (qpp-tuple-group t) fn)
	(fn (qpp-tuple-having t))
	(qpp-map-order (qpp-tuple-order t) fn)
	(qpp-tuple-limit t)
	(qpp-tuple-offset t))))

/* ==================== Pass 1: alias_normalize_pass ==================== */

/* alias_normalize_pass — replace every (get_column alias ti col ci) alias
with its visible occurrence (per visible_occurrence_alias). This drops the
NUL-separator alias provenance left over from derived-table flattening and
picks the visible side of (visible_alias canonical) pairs. */
(define alias_normalize_pass (lambda (t)
	(qpp-apply-to-tuple t normalize_visible_aliases)))

/* ==================== Pass 2: column_resolve_pass ==================== */

/* column_resolve_pass — resolve every (get_column alias ti col ci) in the
tuple so that ti and ci are no longer relevant (the resolver replaces the
expression with canonical casing). `schemas` is a flat assoc list per the
reduce_assoc convention used throughout queryplan.scm:
  (alias1 cols1 alias2 cols2 ...)
where each cols entry is a list of column-defining dicts (each with a "Field"
key, as produced by `show "columns" schema table`).

Pure function: the caller supplies schemas, the pass does no I/O. */
(define column_resolve_pass (lambda (t schemas)
	(qpp-apply-to-tuple t (lambda (expr)
		(canonicalize_columns_scoped expr schemas schemas)))))

/* ==================== Pass 3: derived_table_inline_pass (FAQ §36) ==================== */

/* Per FAQ §36: "derived tables must be inlined as early as possible. materialization
is forbidden. inlining will create a renaming map for all projection columns, so it
automatically prunes unused columns. What's not used will not be inlined."

This pass walks the outer tuple's tables list. For each derived-table entry
  (alias schema sub-7-tuple isOuter joinExpr)
where the sub-tuple is SAFE TO INLINE (no GROUP BY, no LIMIT, no UNION ALL,
no recursive — these would require materialization which §36 forbids), the
pass:
  1. Builds a rename map: every (get_column alias col) in the outer →
     the corresponding expression from the sub's `fields` projections
  2. Replaces the derived entry in the outer's tables list with the sub's
     OWN tables (preserving isOuter / joinExpr semantics — see below)
  3. Merges the sub's WHERE condition into the outer's WHERE
  4. Rewrites all expressions in the outer (fields, condition, group,
     having, order) by applying the rename map

Sub-tuples with GROUP BY stay as derived (FAQ §32: group caches materialize
via make_keytable, which is the standard physical pattern). Other shapes that
prevent inlining: HAVING (needs GROUP), LIMIT/OFFSET (changes row set
semantics), UNION ALL (multiple branches).

For isOuter=true derived tables (LEFT JOIN-ed): inlining is more delicate
because the outer's column references must NULL-extend when no inner row
matches. For now this pass refuses to inline isOuter=true derived tables —
they stay as derived entries (build_queryplan_inner handles the materialization
+ outer-join semantics natively). */

(define qpp-derived-can-inline? (lambda (sub)
	(and (qpp-tuple? sub)
		(or (nil? (qpp-tuple-group sub))
			(equal? (count (qpp-tuple-group sub)) 0))
		(nil? (qpp-tuple-having sub))
		(or (nil? (qpp-tuple-order sub))
			(equal? (count (qpp-tuple-order sub)) 0))
		(nil? (qpp-tuple-limit sub))
		(nil? (qpp-tuple-offset sub)))))

/* qpp-expr-has-wildcard? — true if expr contains a (get_column _ _ "*" _) ref.
SELECT *-style queries can't be inlined because * expansion happens via the
legacy code's schema lookup which needs the derived alias to still be in the
tables list. */
(define qpp-expr-has-wildcard? (lambda (expr) (match expr
	'((symbol get_column) tv ti col ci) (equal? col "*")
	'((quote get_column)  tv ti col ci) (equal? col "*")
	(cons head args) (reduce (coalesceNil args (list)) (lambda (acc a)
		(or acc (qpp-expr-has-wildcard? a))) false)
	false)))

(define qpp-tuple-has-wildcard? (lambda (t) (begin
	(define fs (qpp-fields-to-pairs (qpp-tuple-fields t)))
	(reduce fs (lambda (acc pair) (match pair
		'(name expr) (or acc (qpp-expr-has-wildcard? expr))
		acc)) false))))

/* qpp-build-rename-map — given a derived alias and the sub's fields list,
returns a session whose "map" key holds (alias.col → expr) pairs that should
substitute outer references to the derived table. Per FAQ "renaming map for
all projection columns" — accepts both flat and pair fields shapes. */
(define qpp-build-rename-map (lambda (derived-alias sub-fields) (begin
	(define ren (newsession))
	(ren "map" (list))
	(reduce (qpp-fields-to-pairs sub-fields) (lambda (acc pair) (match pair
		'(name expr) (begin
			(ren "map" (merge (ren "map")
				(list (list (list derived-alias name) expr))))
			acc)
		acc)) nil)
	ren)))

(define qpp-rename-lookup (lambda (ren ref) (begin
	(define entries (ren "map"))
	(reduce entries (lambda (found entry)
		(if (and (nil? found) (equal? (nth entry 0) ref))
			(nth entry 1)
			found))
		nil))))

/* qpp-rewrite-derived-refs — walk an expression tree, replacing every
(get_column derived-alias _ col _) that matches an entry in ren with the
mapped expression. */
(define qpp-rewrite-derived-refs (lambda (expr ren) (match expr
	'((symbol get_column) tv ti col ci)
	(begin
		(define mapped (qpp-rename-lookup ren (list tv col)))
		(if (nil? mapped) expr mapped))
	'((quote get_column) tv ti col ci)
	(begin
		(define mapped (qpp-rename-lookup ren (list tv col)))
		(if (nil? mapped) expr mapped))
	(cons head args)
	(cons head (map (coalesceNil args (list))
		(lambda (a) (qpp-rewrite-derived-refs a ren))))
	expr)))

/* qpp-rewrite-derived-refs-in-fields — preserves the input fields shape
(pairs or flat). The neumann pipeline operates on pairs internally; this
pass sits before lift/unnest/lower so the OUTPUT must be in the same shape
that the subsequent passes expect (pairs). */
(define qpp-rewrite-derived-refs-in-fields (lambda (fields ren)
	(map (qpp-fields-to-pairs fields) (lambda (pair) (match pair
		'(name expr) (list name (qpp-rewrite-derived-refs expr ren))
		pair)))))

(define qpp-rewrite-derived-refs-in-list (lambda (lst ren)
	(if (nil? lst) nil
		(map lst (lambda (e) (qpp-rewrite-derived-refs e ren))))))

(define qpp-rewrite-derived-refs-in-order (lambda (order ren)
	(if (nil? order) nil
		(map order (lambda (item) (match item
			'(expr dir) (list (qpp-rewrite-derived-refs expr ren) dir)
			item))))))

/* qpp-and-cond-merge — combine two WHERE conditions with AND, eliding true. */
(define qpp-and-cond-merge (lambda (a b)
	(if (or (nil? a) (equal? a true) (equal? a (quote true))) b
		(if (or (nil? b) (equal? b true) (equal? b (quote true))) a
			(list (quote and) a b)))))

/* derived_table_inline_pass — the main entry. Walks tables; for each
inlinable derived entry, inlines it. Returns a new 7-tuple.

If the outer query has SELECT * (wildcard ref), inlining is skipped
entirely — wildcard expansion is done by legacy code using the
table-list schema lookup, which needs the derived alias intact. */
(define derived_table_inline_pass (lambda (t) (begin
	/* SELECT * over derived: skip inlining (wildcard expansion happens in
	   the downstream emit code which uses table-list schema lookup; the
	   derived alias must stay so the lookup finds it). */
	(if (qpp-tuple-has-wildcard? t) t (begin
	(define orig-tables (qpp-tuple-tables t))
	(define accumulator (newsession))
	(accumulator "tables" (list))
	(accumulator "cond" (qpp-tuple-condition t))
	(accumulator "ren-map" (list))   /* accumulated rename entries */
	(reduce (coalesceNil orig-tables (list)) (lambda (acc td) (begin
		(if (or (nil? td) (not (list? td)) (< (count td) 3))
			(accumulator "tables" (merge (accumulator "tables") (list td)))
			(begin
				(define derived-alias (nth td 0))
				(define derived-schema (nth td 1))
				(define maybe-sub (nth td 2))
				(define is-outer (if (and (list? td) (>= (count td) 4)) (nth td 3) false))
				(if (or (not (qpp-tuple? maybe-sub))
						(equal? is-outer true)
						(not (qpp-derived-can-inline? maybe-sub)))
					/* Can't inline: keep this table entry as-is. */
					(accumulator "tables" (merge (accumulator "tables") (list td)))
					/* Inlinable derived sub-tuple: */
					(begin
						/* Build rename map from sub's projections under this alias. */
						(define ren-pairs
							(reduce (qpp-fields-to-pairs (qpp-tuple-fields maybe-sub))
								(lambda (rp pair) (match pair
									'(name expr) (merge rp
										(list (list (list derived-alias name) expr)))
									rp)) (list)))
						(accumulator "ren-map"
							(merge (accumulator "ren-map") ren-pairs))
						/* Add sub's tables to outer's tables list. */
						(accumulator "tables"
							(merge (accumulator "tables")
								(coalesceNil (qpp-tuple-tables maybe-sub) (list))))
						/* Merge sub's WHERE into outer's condition. */
						(accumulator "cond"
							(qpp-and-cond-merge (accumulator "cond")
								(qpp-tuple-condition maybe-sub)))))))
		acc)) nil)
	/* Build the combined rename session and rewrite the outer's expressions. */
	(define ren (newsession))
	(ren "map" (accumulator "ren-map"))
	/* Also rewrite joinExpr (5th slot) in any remaining table entry — a
	   LEFT JOIN's ON-clause may reference an inlined derived table's columns
	   that no longer exist after the alias was removed. */
	(define tables-with-rewritten-joinexprs
		(map (accumulator "tables") (lambda (td)
			(if (or (nil? td) (not (list? td)) (< (count td) 5))
				td
				(begin
					(define old-joinexpr (nth td 4))
					(if (nil? old-joinexpr) td
						(list (nth td 0) (nth td 1) (nth td 2) (nth td 3)
							(qpp-rewrite-derived-refs old-joinexpr ren))))))))
	(qpp-rebuild-tuple
		(qpp-tuple-schema t)
		tables-with-rewritten-joinexprs
		(qpp-rewrite-derived-refs-in-fields (qpp-tuple-fields t) ren)
		(qpp-rewrite-derived-refs (accumulator "cond") ren)
		(qpp-rewrite-derived-refs-in-list (qpp-tuple-group t) ren)
		/* Preserve nil for having (legacy distinguishes nil-having from
		   true-having; coalescing to true breaks the having slot). */
		(if (nil? (qpp-tuple-having t)) nil
			(qpp-rewrite-derived-refs (qpp-tuple-having t) ren))
		(qpp-rewrite-derived-refs-in-order (qpp-tuple-order t) ren)
		(qpp-tuple-limit t)
		(qpp-tuple-offset t)))))))
