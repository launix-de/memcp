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

/* qpp-shape-preserving-map-fields — like qpp-map-fields but preserves the
input's flat-or-pairs shape exactly. flat input stays flat; pair input stays
pairs. Critical for sub-tuples that downstream legacy consumers expect in
flat form (e.g. derived sub-tuples carrying window_func expressions). */
(define qpp-shape-preserving-map-fields (lambda (fields fn) (begin
	(define fs (coalesceNil fields '()))
	(if (equal? (count fs) 0)
		fs
		(if (and (list? (nth fs 0)) (equal? (count (nth fs 0)) 2))
			/* PAIRS form: map over pairs */
			(map fs (lambda (pair)
				(match pair
					'(name expr) (list name (fn expr))
					pair)))
			/* FLAT form: stride 2, apply fn to expr positions only */
			(merge
				(map (produceN (/ (count fs) 2))
					(lambda (i)
						(list
							(nth fs (* i 2))
							(fn (nth fs (+ (* i 2) 1))))))))))))

/* ==================== Pass 0.5: alias_disambiguate_pass (v2) ==================== */

/* alias_disambiguate_pass — rename table aliases inside inner scopes
(derived sub-tuples in tables list, and inner_select / inner_select_in /
inner_select_exists markers in expressions) when they collide with an outer
scope's alias. The renamed alias is suffixed with `:N` where N is a fresh
counter so the new alias is provably unique across the whole tree.

V2 fixes the original draft's structural bug: it preserves the field-shape
(flat vs pairs) of every sub-tuple so downstream consumers expecting flat
form (e.g. derived sub-tuples carrying window_func expressions through
legacy build_queryplan_inner) are not corrupted.

Per FAQ §35 "canonical names from physical source columns": renaming
is purely syntactic. The underlying table identity is unchanged. */

(define qpp-alias-counter (newsession))
(qpp-alias-counter "n" 0)
(define qpp-fresh-suffix (lambda () (begin
	(qpp-alias-counter "n" (+ (qpp-alias-counter "n") 1))
	(string (qpp-alias-counter "n")))))

/* qpp-rename-refs-in-expr — walk expr, rewrite get_column tv refs using
rename-map. rename-map is a list of (from-alias to-alias) pairs. */
(define qpp-rename-refs-in-expr (lambda (expr rename-map)
	(if (equal? (count rename-map) 0) expr
		(match expr
			'((symbol get_column) tv ti col ci)
				(begin
					(define new-tv (reduce rename-map (lambda (acc entry) (match entry
						'(from to) (if (and (nil? acc) (equal? from tv)) to acc)
						acc)) nil))
					(if (nil? new-tv) expr
						(list (quote get_column) new-tv ti col ci)))
			'((quote get_column) tv ti col ci)
				(begin
					(define new-tv (reduce rename-map (lambda (acc entry) (match entry
						'(from to) (if (and (nil? acc) (equal? from tv)) to acc)
						acc)) nil))
					(if (nil? new-tv) expr
						(list (quote get_column) new-tv ti col ci)))
			(cons head args) (cons head (map (coalesceNil args '())
				(lambda (a) (qpp-rename-refs-in-expr a rename-map))))
			expr))))

(define qpp-disambiguate-tuple-recursive (lambda (t outer-aliases)
	(if (not (qpp-tuple? t)) t
		(begin
			(define orig-tables (coalesceNil (qpp-tuple-tables t) '()))
			/* Decide renames for each colliding table */
			(define plan (reduce orig-tables (lambda (acc td) (match td
				'(alias tschema ttbl io je)
					(begin
						(define effective-alias (if (nil? alias) ttbl alias))
						(if (has? outer-aliases effective-alias)
							(begin
								(define new-alias (concat (string effective-alias) ":" (qpp-fresh-suffix)))
								(list
									(merge (nth acc 0) (list (list effective-alias new-alias)))
									(merge (nth acc 1) (list (list new-alias tschema ttbl io je)))))
							(list (nth acc 0) (merge (nth acc 1) (list td)))))
				(list (nth acc 0) (merge (nth acc 1) (list td)))))
				(list '() '())))
			(define rename-map (nth plan 0))
			(define renamed-tables-step1 (nth plan 1))
			/* Apply renames to joinExpr in this scope's tables */
			(define renamed-tables-with-je (map renamed-tables-step1 (lambda (td) (match td
				'(alias tschema ttbl io je)
					(list alias tschema ttbl io
						(if (nil? je) nil (qpp-rename-refs-in-expr je rename-map)))
				td))))
			/* New visible set = outer + this scope's aliases */
			(define this-scope-aliases (map renamed-tables-with-je (lambda (td) (match td
				'(alias tschema ttbl io je) (if (nil? alias) ttbl alias)
				nil))))
			(define new-visible (merge outer-aliases (filter this-scope-aliases
				(lambda (a) (not (nil? a))))))
			/* Recurse into derived sub-tuples in tables */
			(define final-tables (map renamed-tables-with-je (lambda (td) (match td
				'(alias tschema ttbl io je)
					(if (qpp-tuple? ttbl)
						(list alias tschema
							(qpp-disambiguate-tuple-recursive ttbl new-visible)
							io je)
						td)
				td))))
			/* Rewrite expression slots: apply renames + recurse into inner_select* */
			(define rewrite-expr (lambda (expr)
				(qpp-disambiguate-expr-recursive
					(qpp-rename-refs-in-expr expr rename-map)
					new-visible)))
			/* Build with SHAPE-PRESERVING field mapping */
			(qpp-rebuild-tuple
				(qpp-tuple-schema t)
				final-tables
				(qpp-shape-preserving-map-fields (qpp-tuple-fields t) rewrite-expr)
				(rewrite-expr (qpp-tuple-condition t))
				(qpp-map-group (qpp-tuple-group t) rewrite-expr)
				(rewrite-expr (qpp-tuple-having t))
				(qpp-map-order (qpp-tuple-order t) rewrite-expr)
				(qpp-tuple-limit t)
				(qpp-tuple-offset t))))))

(define qpp-disambiguate-expr-recursive (lambda (expr outer-aliases)
	(match expr
		(cons sym args)
			(begin
				(define is-scalar (match sym
					(symbol inner_select)        true
					(quote inner_select)         true
					'(quote inner_select)        true
					'inner_select                true
					false))
				(define is-in (match sym
					(symbol inner_select_in)     true
					(quote inner_select_in)      true
					'(quote inner_select_in)     true
					'inner_select_in             true
					false))
				(define is-exists (match sym
					(symbol inner_select_exists) true
					(quote inner_select_exists)  true
					'(quote inner_select_exists) true
					'inner_select_exists         true
					false))
				(if is-scalar
					(if (>= (count args) 1)
						(list sym (qpp-disambiguate-tuple-recursive (nth args 0) outer-aliases))
						expr)
				(if is-in
					(if (>= (count args) 2)
						(list sym
							(qpp-disambiguate-expr-recursive (nth args 0) outer-aliases)
							(qpp-disambiguate-tuple-recursive (nth args 1) outer-aliases))
						expr)
				(if is-exists
					(if (>= (count args) 1)
						(list sym (qpp-disambiguate-tuple-recursive (nth args 0) outer-aliases))
						expr)
					(cons sym (map (coalesceNil args '())
						(lambda (a) (qpp-disambiguate-expr-recursive a outer-aliases))))))))
		expr)))

(define alias_disambiguate_pass (lambda (t)
	(qpp-disambiguate-tuple-recursive t '())))

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

Pure function: the caller supplies schemas, the pass does no I/O.

Single-scope variant. For correlated queries with nested inner_select markers,
use [column_resolve_scoped_pass] which recurses into sub-tuples with their own
local schemas — that variant is the architectural fix for nil-tv refs leaking
across scope boundaries. */
(define column_resolve_pass (lambda (t schemas)
	(qpp-apply-to-tuple t (lambda (expr)
		(canonicalize_columns_scoped expr schemas schemas)))))

/* qpp-schemas-from-tables — build a schemas assoc list from a 7-tuple's tables
list. Real tables resolve via `(show schema tbl)`; derived sub-tuples expose
their projection field names as columns (Type "any").

Returns a flat assoc list (alias1 cols1 alias2 cols2 ...) suitable for
reduce_assoc and canonicalize_columns_scoped. */
(define qpp-schemas-from-tables (lambda (tables)
	(reduce (coalesceNil tables '()) (lambda (acc td) (match td
		'(alias tschema ttbl io je)
			(begin
				(define resolved-alias (if (nil? alias) ttbl alias))
				(if (qpp-tuple? ttbl)
					/* Derived sub-tuple: project field names as columns. */
					(merge acc (list resolved-alias
						(map (qpp-fields-to-pairs (qpp-tuple-fields ttbl))
							(lambda (p) (list "Field" (nth p 0) "Type" "any")))))
					/* Real table name (string or symbol): use get_schema which
					   handles INFORMATION_SCHEMA + falls through to (show schema tbl)
					   for normal tables. Wrapped in try in case the table doesn't
					   yet exist (CREATE TABLE flows, EXPLAIN) — drop the entry
					   rather than poisoning the resolver. */
					(begin
						(define cols (try (lambda () (get_schema tschema ttbl))
							(lambda (e) '())))
						(if (or (nil? cols) (equal? cols '())) acc
							(merge acc (list resolved-alias cols))))))
		acc)) '())))

/* qpp-resolve-expr-scoped — resolve (get_column …) refs in expr with
local+visible schemas, recursing into inner_select / inner_select_in /
inner_select_exists markers with the sub-tuple's OWN local schemas and
the caller's visible schemas extended by the caller's locals (= SQL scope
nesting per ISO standard). */
/* qpp-unresolved-nil-ref? — true if expr is `(get_column nil … … …)` with
   any of ti/ci flags still true (= unresolved). Used to detect fallback need
   after a local-only resolution attempt. */
(define qpp-unresolved-nil-ref? (lambda (expr) (match expr
	'((symbol get_column) tv ti col ci) (and (nil? tv) (or ti ci))
	'((quote get_column)  tv ti col ci) (and (nil? tv) (or ti ci))
	false)))

(define qpp-resolve-expr-scoped (lambda (expr local-schemas visible-schemas) (begin
	(define is-scalar-marker? (lambda (sym) (match sym
		(symbol inner_select)       true
		(quote inner_select)        true
		'(quote inner_select)       true
		'inner_select               true
		false)))
	(define is-in-marker? (lambda (sym) (match sym
		(symbol inner_select_in)    true
		(quote inner_select_in)     true
		'(quote inner_select_in)    true
		'inner_select_in            true
		false)))
	(define is-exists-marker? (lambda (sym) (match sym
		(symbol inner_select_exists) true
		(quote inner_select_exists)  true
		'(quote inner_select_exists) true
		'inner_select_exists         true
		false)))
	(define is-getcol? (lambda (sym) (match sym
		(symbol get_column) true
		(quote get_column)  true
		'(quote get_column) true
		'get_column         true
		false)))
	(match expr
		(cons sym args)
			(if (is-getcol? sym)
				/* Atomic — resolve the WHOLE get_column expression in one shot
				   using canonicalize_columns_scoped (which understands the
				   (get_column alias ti col ci) shape and uses ti/ci to do
				   case-insensitive lookups via local + visible schemas).

				   canonicalize_columns_scoped only resolves UNQUALIFIED refs
				   against local schemas (so recursive scopes don't accidentally
				   bind to outer aliases). For SQL scope-fallback semantics —
				   an unqualified inner ref that doesn't exist in local tables
				   should resolve to the outer scope — fall through to a second
				   resolve against visible-schemas as both local and visible. */
				(begin
					(define first-try
						(canonicalize_columns_scoped expr local-schemas visible-schemas))
					(if (qpp-unresolved-nil-ref? first-try)
						(canonicalize_columns_scoped first-try visible-schemas visible-schemas)
						first-try))
			(if (is-scalar-marker? sym)
				/* (inner_select sub-7tuple): recurse with sub's scope. */
				(list sym (qpp-resolve-tuple-scoped (nth args 0) visible-schemas))
			(if (is-in-marker? sym)
				/* (inner_select_in lhs sub): lhs in caller scope, sub recurses. */
				(list sym
					(qpp-resolve-expr-scoped (nth args 0) local-schemas visible-schemas)
					(qpp-resolve-tuple-scoped (nth args 1) visible-schemas))
			(if (is-exists-marker? sym)
				(list sym (qpp-resolve-tuple-scoped (nth args 0) visible-schemas))
			/* Default: recurse into args (operator call). */
			(cons sym (map (coalesceNil args '())
				(lambda (a) (qpp-resolve-expr-scoped a local-schemas visible-schemas))))))))
		/* Leaf atom: nothing to resolve. */
		expr))))

/* qpp-resolve-tuple-scoped — recursive scope-aware column resolution for a
sub 7-tuple. Builds the sub's local schemas, merges with the caller's
visible-schemas to form the new visible (per SQL scope rules: outer
qualified refs are visible inside, unqualified inner refs match local
schemas first). */
(define qpp-resolve-tuple-scoped (lambda (t outer-schemas)
	(if (not (qpp-tuple? t)) t
		(begin
			(define local-schemas (qpp-schemas-from-tables (qpp-tuple-tables t)))
			(define visible-schemas (merge outer-schemas local-schemas))
			(qpp-apply-to-tuple t (lambda (expr)
				(qpp-resolve-expr-scoped expr local-schemas visible-schemas)))))))

/* column_resolve_scoped_pass — the architecturally-correct column resolver.
Drop-in replacement for column_resolve_pass that recurses into inner_select*
markers with proper scope nesting. At the top level outer-schemas is the
empty list. */
(define column_resolve_scoped_pass (lambda (t outer-schemas)
	(qpp-resolve-tuple-scoped t (coalesceNil outer-schemas '()))))

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
