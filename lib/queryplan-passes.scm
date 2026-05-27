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

/* qpp-map-fields — apply fn to every projection expression in a fields list.
fields shape: ((name expr) (name expr) ...) */
(define qpp-map-fields (lambda (fields fn)
	(map (coalesceNil fields '()) (lambda (pair) (match pair
		'(name expr) (list name (fn expr))
		pair)))))

/* qpp-map-order — apply fn to every order expression. order: ((expr dir) ...) */
(define qpp-map-order (lambda (order fn)
	(map (coalesceNil order '()) (lambda (item) (match item
		'(expr dir) (list (fn expr) dir)
		item)))))

/* qpp-map-group — apply fn to every group-by expression. group: (expr ...) */
(define qpp-map-group (lambda (group fn)
	(map (coalesceNil group '()) fn)))

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
