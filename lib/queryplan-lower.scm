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
== Layer 4 — lower_to_scans_pass: qpir tree → 7-tuple ==

Per the FAQ: "unnesting a query does NOT mean no scalar subselect. it just
means: untangle the query first, then apply join_reorder optimizations and
then build nested selects from it again."

This pass takes the holistically-unnested qpir tree (no qpir-dep-join,
F(root)=∅) and rewrites it back into a SINGLE 7-tuple that the existing
build_queryplan_inner already knows how to compile via the standard scan /
join / groupby paths. The 7-tuple may contain DERIVED TABLES (sub-7-tuples
inside the tables list) for qpir-groupby outputs — those follow the parser
shape `(alias schema sub-7-tuple false nil)` per lib/sql-parser.scm
tabledef productions.

Per FAQ: "groups are HARD borders in queryplan compilation — never cross
them inline." So a qpir-groupby in the tree becomes its own 7-tuple
universe (a derived table) that build_queryplan_inner materializes via
make_keytable + createcolumn.

Per FAQ "derived tables must be inlined as early as possible" — this only
applies to non-grouped derived tables. Grouped derived tables (from a
qpir-groupby) must materialize because GROUP BY is a hard border.

Operator → 7-tuple rules:

qpir-leaf 7tuple                  → 7tuple

qpir-select pred child            → child-tuple with pred AND-ed into
its condition

qpir-map projs child              → child-tuple with projs replacing
its fields

qpir-groupby keys aggs having child → child-tuple wrapped as a 7-tuple
with keys in `group`, aggs in
`fields`, having in `having`.
Returns a SEPARATE 7-tuple that
callers wrap as a derived table.

qpir-join inner pred left right   → merged 7-tuple where:
tables = left.tables ++ right-table-entry
condition = AND(left.cond, right.cond, pred)
fields = left.fields (outer wins)
right-table-entry depends on whether
the right is a leaf (use its tables
directly) or a groupby (wrap as
derived table with rhs-alias).

qpir-join with rhs-alias          → right wrapped as derived table aliased
rhs-alias. Outer's column refs to
rhs-alias.<col> resolve via the
derived-table's projection.

Phase 1 of this module supports the canonical correlated-SUM shape after
unnest:
qpir-join inner (po.k = pi.k) rhs-alias=sq_1
qpir-leaf po
qpir-groupby [pi.k] [(value SUM)] nil
qpir-leaf pi
→
7-tuple {tables: [po, (sq_1 schema <group-7-tuple> false nil)],
fields: [(id po.id) (total sq_1.value)],
condition: (po.k = sq_1.k)}
where <group-7-tuple> is the SELECT pi.k, SUM(pi.amount) FROM pi GROUP BY pi.k

Phase 2+ adds: qpir-window, qpir-union, qpir-iterate, multi-level groupby
nesting, derived-table inlining for non-grouped sub-queries.
*/

/* ==================== Utility helpers ==================== */

/* qpu-low-and-cond — combine two predicates with AND, eliding trivial true. */
(define qpu-low-and-cond (lambda (a b)
	(if (or (nil? a) (equal? a true) (equal? a (quote true))) b
		(if (or (nil? b) (equal? b true) (equal? b (quote true))) a
			(list (quote and) a b)))))

(define qpu-low-equality-op? (lambda (op)
	(or (equal? op (quote equal?))
		(equal? op (symbol equal?))
		(equal? op (quote equal??))
		(equal? op (symbol equal??)))))

(define qpu-low-same-equality? (lambda (a b)
	(match a
		'(aop al ar)
		(if (qpu-low-equality-op? aop)
			(match b
				'(bop bl br)
				(and (qpu-low-equality-op? bop)
					(or (and (equal? al bl) (equal? ar br))
						(and (equal? al br) (equal? ar bl))))
				false)
			false)
		false)))

(define qpu-low-remove-conjunct (lambda (cond target)
	(if (or (nil? cond) (equal? cond true) (equal? cond (quote true)))
		cond
		(if (or (equal? cond target) (qpu-low-same-equality? cond target))
			true
			(match cond
				'((symbol and) a b)
				(qpu-low-and-cond
					(qpu-low-remove-conjunct a target)
					(qpu-low-remove-conjunct b target))
				'((quote and) a b)
				(qpu-low-and-cond
					(qpu-low-remove-conjunct a target)
					(qpu-low-remove-conjunct b target))
				cond)))))

/* qpu-low-key-projection — turn a group-by key expression into a (name expr)
projection pair. If the key is a bare (get_column tv ti col ci), the projection
name is the col. Otherwise we synthesize a key_N name. */
(define qpu-low-key-counter (newsession))
(qpu-low-key-counter "n" 0)
(define qpu-low-fresh-key-name (lambda () (begin
	(qpu-low-key-counter "n" (+ (qpu-low-key-counter "n") 1))
	(concat "key_" (string (qpu-low-key-counter "n"))))))

(define qpu-low-key-projection (lambda (key-expr)
	(match key-expr
		'((symbol get_column) tv ti col ci) (list col key-expr)
		'((quote get_column)  tv ti col ci) (list col key-expr)
		(list (qpu-low-fresh-key-name) key-expr))))

/* qpu-low-rewrite-refs — rewrite every (get_column tv …) in expr where
tv matches one of `from-aliases` to use `to-alias` instead. Used when a
qpir-join with rhs-alias wraps the right as a derived table — outer refs
to the right-side tables get retargeted to the rhs-alias. */
(define qpu-low-rewrite-refs (lambda (expr from-aliases to-alias)
	(match expr
		'((symbol get_column) tv ti col ci)
		(if (has? from-aliases tv)
			(list (quote get_column) to-alias false
				(if (and (string? to-alias) (>= (strlen to-alias) 3)
					(equal? (substr to-alias 0 3) "sq_"))
					(qpu-low-kt-ref-source-col col)
					col)
				false)
			expr)
		'((quote get_column) tv ti col ci)
		(if (has? from-aliases tv)
			(list (quote get_column) to-alias false
				(if (and (string? to-alias) (>= (strlen to-alias) 3)
					(equal? (substr to-alias 0 3) "sq_"))
					(qpu-low-kt-ref-source-col col)
					col)
				false)
			expr)
		(cons head args)
		(cons head (map (coalesceNil args (list))
			(lambda (a) (qpu-low-rewrite-refs a from-aliases to-alias))))
		expr)))

/* qpu-low-rewrite-projections — apply qpu-low-rewrite-refs to every
projection in a fields list. */
(define qpu-low-rewrite-projections (lambda (projections from-aliases to-alias)
	(map (coalesceNil projections (list)) (lambda (pair) (match pair
		'(name expr) (list name (qpu-low-rewrite-refs expr from-aliases to-alias))
		pair)))))

/* qpu-low-collect-refs-for-aliases — walk expr, return all `(tv col)` pairs
for `(get_column tv ti col ci)` where tv ∈ aliases. */
(define qpu-low-collect-refs-for-aliases (lambda (expr aliases) (begin
	(match expr
		'((symbol get_column) tv ti col ci)
		(if (and (not (nil? tv)) (has? aliases tv)) (list (list tv col)) '())
		'((quote get_column)  tv ti col ci)
		(if (and (not (nil? tv)) (has? aliases tv)) (list (list tv col)) '())
		(cons head args)
		(reduce (coalesceNil args '()) (lambda (acc a)
			(merge acc (qpu-low-collect-refs-for-aliases a aliases))) '())
		'()))))

/* qpu-low-fields-find-by-expr — find the field whose VALUE expression matches
`(get_column tv false col false)`. Returns the field NAME or nil. */
(define qpu-low-fields-find-by-expr (lambda (fields tv col)
	(reduce (coalesceNil fields '()) (lambda (acc pair) (match pair
		'(n e) (if (not (nil? acc)) acc
			(match e
				'((symbol get_column) etv eti ecol eci)
				(if (and (equal? etv tv) (equal? ecol col)) n acc)
				'((quote get_column) etv eti ecol eci)
				(if (and (equal? etv tv) (equal? ecol col)) n acc)
				acc))
		acc)) nil)))

(define qpu-low-fields-has-name (lambda (fields name)
	(reduce (coalesceNil fields '()) (lambda (acc pair) (match pair
		'(n e) (or acc (equal? n name))
		acc)) false)))

(define qpu-low-fields-find-by-name (lambda (fields name)
	(reduce (coalesceNil fields '()) (lambda (acc pair) (match pair
		'(n e) (if (and (nil? acc) (equal? n name)) n acc)
		acc)) nil)))

/* qpu-low-find-deep-alias-in-tables — for an alias name NOT in the top-level
table aliases, find which top-level DERIVED table's sub-tuple contains it.
Returns the derived-alias name, or nil if alias isn't reachable through any
of the top-level deriveds. Used by qpu-low-ensure-join-key-fields to cascade
__kt projections through nested rhs-aliased wrappers (FAQ §42 step 2/2). */
(define qpu-low-tables-contain-alias-deep? (lambda (tables alias)
	(reduce (coalesceNil tables '()) (lambda (acc td) (match td
		'(td-alias td-schema td-tname td-isOuter td-jE)
		(or acc (equal? td-alias alias)
			(and (qpp-tuple? td-tname)
				(qpu-low-tables-contain-alias-deep? (qpp-tuple-tables td-tname) alias)))
		acc)) false)))

(define qpu-low-find-deep-alias-in-tables (lambda (tables alias)
	(reduce (coalesceNil tables '()) (lambda (acc td) (match td
		'(td-alias td-schema td-tname td-isOuter td-jE)
		(if (not (nil? acc)) acc
			(if (qpp-tuple? td-tname)
				(if (qpu-low-tables-contain-alias-deep? (qpp-tuple-tables td-tname) alias)
					td-alias
					nil)
				nil))
		acc)) nil)))

(define qpu-low-top-alias-derived? (lambda (tables alias)
	(reduce (coalesceNil tables '()) (lambda (acc td) (match td
		'(td-alias td-schema td-tname td-isOuter td-jE)
		(or acc (and (equal? td-alias alias) (qpp-tuple? td-tname)))
		acc)) false)))

(define qpu-low-top-alias-materialized? (lambda (tables alias)
	(reduce (coalesceNil tables '()) (lambda (acc td) (match td
		'(td-alias td-schema td-tname td-isOuter td-jE)
		(or acc (and (equal? td-alias alias)
			(or (materialized-source? td-tname)
				(strlike (string td-tname) "%__mat:%"))))
		acc)) false)))

(define qpu-low-scan-tagged-table? (lambda (tbl)
	(match tbl
		'(scan-tagged-table _ _ _ _ _ _) true
		'(scan-tagged-table _ _ _ _ _ _ _) true
		'((symbol scan-tagged-table) _ _ _ _ _ _) true
		'((symbol scan-tagged-table) _ _ _ _ _ _ _) true
		'((quote scan-tagged-table) _ _ _ _ _ _) true
		'((quote scan-tagged-table) _ _ _ _ _ _ _) true
		false)))

(define qpu-low-strip-artificial-scalar-tags (lambda (tuple)
	(if (not (qpp-tuple? tuple)) tuple
		(qpp-rebuild-tuple
			(qpp-tuple-schema tuple)
			(map (coalesceNil (qpp-tuple-tables tuple) '()) (lambda (td) (match td
				'(td-alias td-schema td-tname td-isOuter td-jE)
				(if (and (qpu-low-scan-tagged-table? td-tname)
					(equal? (coalesceNil (scan_tagged_table_order td-tname) '()) '())
					(equal? (scan_tagged_table_limit td-tname) 2)
					(nil? (scan_tagged_table_offset td-tname))
					(equal? (scan_tagged_table_partition_cols td-tname) 0)
					(or (equal? (scan_tagged_table_once_limit td-tname) 2)
						(equal? (scan_tagged_table_once_limit td-tname) 1)))
					(list td-alias td-schema (scan_tagged_table_base td-tname) td-isOuter td-jE)
					td)
				td)))
			(qpp-tuple-fields tuple)
			(qpp-tuple-condition tuple)
			(qpp-tuple-group tuple)
			(qpp-tuple-having tuple)
			(qpp-tuple-order tuple)
			(qpp-tuple-limit tuple)
			(qpp-tuple-offset tuple)))))

(define qpu-low-strip-scan-tags-recursive (lambda (tuple)
	(if (not (qpp-tuple? tuple)) tuple
		(qpp-rebuild-tuple
			(qpp-tuple-schema tuple)
			(map (coalesceNil (qpp-tuple-tables tuple) '()) (lambda (td) (match td
				'(td-alias td-schema td-tname td-isOuter td-jE)
				(list td-alias td-schema
					(if (qpu-low-scan-tagged-table? td-tname)
						(scan_tagged_table_base td-tname)
						(if (qpp-tuple? td-tname)
							(qpu-low-strip-scan-tags-recursive td-tname)
							td-tname))
					td-isOuter td-jE)
				td)))
			(qpp-tuple-fields tuple)
			(qpp-tuple-condition tuple)
			(qpp-tuple-group tuple)
			(qpp-tuple-having tuple)
			(qpp-tuple-order tuple)
			(qpp-tuple-limit tuple)
			(qpp-tuple-offset tuple)))))

(define qpu-low-strip-nested-scan-tags (lambda (tuple)
	(if (not (qpp-tuple? tuple)) tuple
		(qpp-rebuild-tuple
			(qpp-tuple-schema tuple)
			(map (coalesceNil (qpp-tuple-tables tuple) '()) (lambda (td) (match td
				'(td-alias td-schema td-tname td-isOuter td-jE)
				(list td-alias td-schema
					(if (qpp-tuple? td-tname)
						(qpu-low-strip-scan-tags-recursive td-tname)
						td-tname)
					td-isOuter td-jE)
				td)))
			(qpp-tuple-fields tuple)
			(qpp-tuple-condition tuple)
			(qpp-tuple-group tuple)
			(qpp-tuple-having tuple)
			(qpp-tuple-order tuple)
			(qpp-tuple-limit tuple)
			(qpp-tuple-offset tuple)))))

/* qpu-low-ensure-nested-derived-projects — for a `(tv col)` ref where `tv`
lives INSIDE one of right-tables's top-level deriveds (alias=derived-alias),
ensure that derived's sub-tuple projects (tv col) under some name. Returns
(updated-tables, name-in-derived) — the caller wraps this with a passthrough
projection at the outer level. */
(define qpu-low-ensure-nested-derived-projects (lambda (tables derived-alias tv col) (begin
	(define result-name (newsession))
	(result-name "n" nil)
	(define updated-tables (map (coalesceNil tables '()) (lambda (td) (match td
		'(td-alias td-schema td-tname td-isOuter td-jE)
		(if (and (equal? td-alias derived-alias) (qpp-tuple? td-tname))
			(begin
				(define sub td-tname)
				(define sub-fields (qpp-fields-to-pairs (qpp-tuple-fields sub)))
				(define source-col (qpu-low-kt-ref-source-col col))
				(define exact-existing-name (coalesce
					(qpu-low-fields-find-by-expr sub-fields tv col)
					(qpu-low-fields-find-by-name sub-fields col)))
				(define source-existing-name
					(if (equal? source-col col) nil
						(coalesce
							(qpu-low-fields-find-by-expr sub-fields tv source-col)
							(qpu-low-fields-find-by-name sub-fields source-col))))
				(define existing-name (coalesce exact-existing-name source-existing-name))
				(if (not (nil? existing-name))
					(begin
						(result-name "n" existing-name)
						(list td-alias td-schema (qpu-low-strip-artificial-scalar-tags sub) td-isOuter td-jE))
					(begin
						(define synthesized (if (equal? source-col col)
							(concat "__kt_" col)
							col))
						(define unique-name (qpu-low-unique-projection-name
							synthesized sub-fields))
						(result-name "n" unique-name)
						(define sub-aliases (map (coalesceNil (qpp-tuple-tables sub) '())
							(lambda (itd) (if (or (nil? itd) (< (count itd) 1))
								nil (nth itd 0)))))
						(define nested-derived
							(if (has? sub-aliases tv)
								nil
								(qpu-low-find-deep-alias-in-tables (qpp-tuple-tables sub) tv)))
						(define nested-result
							(if (nil? nested-derived)
								nil
								(qpu-low-ensure-nested-derived-projects
									(qpp-tuple-tables sub) nested-derived tv col)))
						(define passthrough-col
							(if (nil? nested-result)
								(if (and (not (equal? source-col col)) (nil? source-existing-name))
									col
									source-col)
								(nth nested-result 1)))
						(define passthrough-tv
							(if (nil? nested-result) tv nested-derived))
						(define new-sub-fields (merge sub-fields
							(list (list unique-name
								(list (quote get_column) passthrough-tv false passthrough-col false)))))
						(define new-sub (qpp-rebuild-tuple
							(qpp-tuple-schema sub)
							(if (nil? nested-result)
								(qpp-tuple-tables sub)
								(nth nested-result 0))
							new-sub-fields
							(qpp-tuple-condition sub)
							(qpp-tuple-group sub)
							(qpp-tuple-having sub)
							(qpp-tuple-order sub)
							(qpp-tuple-limit sub)
							(qpp-tuple-offset sub)))
						(list td-alias td-schema (qpu-low-strip-artificial-scalar-tags new-sub) td-isOuter td-jE))))
			td)
		td))))
	(list updated-tables (result-name "n")))))

(define qpu-low-kt-ref-source-col (lambda (col)
	(if (and (string? col) (>= (strlen col) 5)
		(equal? (substr col 0 5) "__kt_"))
		(substr col 5 (- (strlen col) 5))
		col)))

(define qpu-low-can-normalize-kt-ref (lambda (fields tv col)
	(begin
		(define source-col (qpu-low-kt-ref-source-col col))
		(and (not (equal? source-col col))
			(or (not (nil? (qpu-low-fields-find-by-expr fields tv source-col)))
				(and
					(not (qpu-low-fields-has-name fields col))
					(qpu-low-fields-has-name fields source-col)))))))

(define qpu-low-normalize-kt-refs-in-expr (lambda (expr aliases fields)
	(match expr
		'((symbol get_column) tv ti col ci)
		(if (and (has? aliases tv) (qpu-low-can-normalize-kt-ref fields tv col))
			(list (quote get_column) tv false (qpu-low-kt-ref-source-col col) false)
			expr)
		'((quote get_column) tv ti col ci)
		(if (and (has? aliases tv) (qpu-low-can-normalize-kt-ref fields tv col))
			(list (quote get_column) tv false (qpu-low-kt-ref-source-col col) false)
			expr)
		(cons head args)
		(cons head (map (coalesceNil args '())
			(lambda (a) (qpu-low-normalize-kt-refs-in-expr a aliases fields))))
		expr)))

(define qpu-low-normalize-kt-refs-in-fields (lambda (fields aliases)
	(map (coalesceNil fields '()) (lambda (pair) (match pair
		'(name expr) (list name (qpu-low-normalize-kt-refs-in-expr
			expr aliases fields))
		pair)))))

/* qpu-low-ensure-join-key-fields — for every `(tv col)` referenced by
`join-pred`, ensure the right-tuple's fields list projects it under SOME
name. Returns a pair (updated-right-tuple, rename-map) where rename-map is
an assoc list of ((tv col) projected-name) so the caller can rewrite
join-pred refs to use the canonical projected name (which may differ from
`col` if `col` was already taken by a different expression).

Three cases per ref:
- tv ∈ right-source-aliases (direct base or derived at top level):
synthesize __kt_<col> projection at top level
- tv lives INSIDE a top-level derived's sub-tuple (FAQ §42 doubly-nested
cascade): recursively ensure the nested derived projects (tv col),
then add a passthrough projection at the top level that exposes it
- tv unreachable (ancestor outer-ref): skip — outer wrapper handles it

Without the cascade case, join-pred retargeting `(r src)` → `(sq_N src)`
produces a reference to a non-existent field — sq_N's sub-tuple doesn't
project r.src at its boundary even though r is reachable inside.

Naming: synthesized name `__kt_<col>` (suffixed if collision). */
(define qpu-low-ensure-join-key-fields (lambda (right-tuple join-pred right-source-aliases)
	(begin
		/* Collect ALL refs in join-pred (not just those in right-source-aliases) —
		the cascade case needs to see refs to deeper-nested aliases too. */
		(define all-refs (qpir-expr-column-refs join-pred))
		(define existing-fields (qpp-fields-to-pairs (qpp-tuple-fields right-tuple)))
		(define top-tables (qpp-tuple-tables right-tuple))
		/* Plan: walk every ref, classify (direct / cascaded / skip), accumulate
		(current-tables, added-projections, rename-map). current-tables may
		change for cascaded refs because nested deriveds get updated in place. */
		(define plan (reduce all-refs (lambda (acc ref) (match ref
			'(tv col)
			(if (has? right-source-aliases tv)
				/* Direct case (existing behavior) */
				(begin
					(define source-col (qpu-low-kt-ref-source-col col))
					(define all-fields (merge existing-fields (nth acc 1)))
					(define exact-existing-name
						(qpu-low-fields-find-by-expr all-fields tv col))
					(define source-existing-name
						(if (equal? source-col col) nil
							(qpu-low-fields-find-by-expr all-fields tv source-col)))
					(if (not (nil? exact-existing-name))
						(list (nth acc 0) (nth acc 1) (merge (nth acc 2)
							(list (list (list tv col) exact-existing-name))))
						(if (not (nil? source-existing-name))
							(if (qpu-low-top-alias-materialized? (nth acc 0) tv)
								(list (nth acc 0) (nth acc 1) (merge (nth acc 2)
									(list (list (list tv col) source-existing-name))))
								(begin
									(define unique-name (qpu-low-unique-projection-name
										col all-fields))
									(list
										(nth acc 0)
										(merge (nth acc 1)
											(list (list unique-name
												(list (quote get_column) tv false source-col false))))
										(merge (nth acc 2)
											(list (list (list tv col) unique-name))))))
							(begin
								(define source-is-materialized
									(qpu-low-top-alias-materialized? (nth acc 0) tv))
								(define passthrough-col
									(if source-is-materialized
										source-col
										(if (and
											(qpu-low-top-alias-derived? (nth acc 0) tv)
											(not (equal? source-col col))
											(nil? source-existing-name))
											col
											source-col)))
								(define synthesized (if source-is-materialized
									source-col
									(if (equal? source-col col)
										(concat "__kt_" col)
										col)))
								(define unique-name (qpu-low-unique-projection-name
									synthesized all-fields))
								(list
									(nth acc 0)
									(merge (nth acc 1)
										(list (list unique-name
											(list (quote get_column) tv false passthrough-col false))))
									(merge (nth acc 2)
										(list (list (list tv col) unique-name))))))))
				/* Not a direct ref — check if tv is inside a top-level derived
				(FAQ §42 cascade case). */
				(begin
					(define deep-derived
						(qpu-low-find-deep-alias-in-tables (nth acc 0) tv))
					(if (nil? deep-derived)
						/* Unreachable from this subtree — ancestor outer ref. */
						acc
						(begin
							(define ensure-result (qpu-low-ensure-nested-derived-projects
								(nth acc 0) deep-derived tv col))
							(define updated-tables (nth ensure-result 0))
							(define name-in-derived (nth ensure-result 1))
							/* Add passthrough projection at top level. */
							(define synthesized (concat "__kt_" col))
							(define unique-name (qpu-low-unique-projection-name
								synthesized (merge existing-fields (nth acc 1))))
							(list
								updated-tables
								(merge (nth acc 1)
									(list (list unique-name
										(list (quote get_column) deep-derived
											false name-in-derived false))))
								(merge (nth acc 2)
									(list (list (list tv col) unique-name))))))))
			acc)) (list top-tables (list) (list))))
		(define final-tables (nth plan 0))
		(define added-projections (nth plan 1))
		(define rename-map (nth plan 2))
		(define merged-fields (merge existing-fields added-projections))
		(define normalized-fields (qpu-low-normalize-kt-refs-in-fields
			merged-fields right-source-aliases))
		(define normalized-cond (qpu-low-normalize-kt-refs-in-expr
			(qpp-tuple-condition right-tuple) right-source-aliases merged-fields))
		(list
			(if (and (equal? (count added-projections) 0)
				(equal? final-tables top-tables))
				(qpp-rebuild-tuple
					(qpp-tuple-schema right-tuple)
					final-tables
					normalized-fields
					normalized-cond
					(qpp-tuple-group right-tuple)
					(qpp-tuple-having right-tuple)
					(qpp-tuple-order right-tuple)
					(qpp-tuple-limit right-tuple)
					(qpp-tuple-offset right-tuple))
				(qpp-rebuild-tuple
					(qpp-tuple-schema right-tuple)
					final-tables
					normalized-fields
					normalized-cond
					(qpp-tuple-group right-tuple)
					(qpp-tuple-having right-tuple)
					(qpp-tuple-order right-tuple)
					(qpp-tuple-limit right-tuple)
					(qpp-tuple-offset right-tuple)))
			rename-map))))

/* qpu-low-unique-projection-name — given a candidate name and an existing
fields list, return either the candidate or `candidate_N` so the result is
unique. */
(define qpu-low-unique-projection-name (lambda (name fields)
	(begin
		(define taken? (lambda (n) (reduce (coalesceNil fields '()) (lambda (acc p) (match p
			'(fn fe) (or acc (equal? fn n))
			acc)) false)))
		(if (not (taken? name)) name
			(begin
				(define n 1)
				(define candidate (concat name "_" (string n)))
				(reduce '(2 3 4 5 6 7 8 9) (lambda (acc i)
					(if (taken? candidate)
						(begin
							(define candidate (concat name "_" (string i)))
							acc)
						acc)) nil)
				candidate)))))

/* qpu-low-rewrite-by-renames — rewrite join-pred refs using rename-map.
Maps `(get_column tv ti col ci)` → `(get_column to-alias false renamed-col false)`
when the pair (tv col) appears in rename-map. */
(define qpu-low-rewrite-by-renames (lambda (expr rename-map to-alias)
	(match expr
		'((symbol get_column) tv ti col ci)
		(begin
			(define renamed (reduce rename-map (lambda (acc entry) (match entry
				'(refpair newname) (match refpair
					'(rtv rcol)
					(if (and (equal? rtv tv) (equal? rcol col)) newname acc)
					acc)
				acc)) nil))
			(if (nil? renamed) expr
				(list (quote get_column) to-alias false renamed false)))
		'((quote get_column) tv ti col ci)
		(begin
			(define renamed (reduce rename-map (lambda (acc entry) (match entry
				'(refpair newname) (match refpair
					'(rtv rcol)
					(if (and (equal? rtv tv) (equal? rcol col)) newname acc)
					acc)
				acc)) nil))
			(if (nil? renamed) expr
				(list (quote get_column) to-alias false renamed false)))
		(cons head args)
		(cons head (map (coalesceNil args '())
			(lambda (a) (qpu-low-rewrite-by-renames a rename-map to-alias))))
		expr)))

(define qpu-low-rewrite-cols-by-renames (lambda (expr rename-map)
	(match expr
		'((symbol get_column) tv ti col ci)
		(begin
			(define renamed (reduce rename-map (lambda (acc entry) (match entry
				'(refpair newname) (match refpair
					'(rtv rcol)
					(if (and (equal? rtv tv) (equal? rcol col)) newname acc)
					acc)
				acc)) nil))
			(if (nil? renamed) expr
				(list (quote get_column) tv false renamed false)))
		'((quote get_column) tv ti col ci)
		(begin
			(define renamed (reduce rename-map (lambda (acc entry) (match entry
				'(refpair newname) (match refpair
					'(rtv rcol)
					(if (and (equal? rtv tv) (equal? rcol col)) newname acc)
					acc)
				acc)) nil))
			(if (nil? renamed) expr
				(list (quote get_column) tv false renamed false)))
		(cons head args)
		(cons head (map (coalesceNil args '())
			(lambda (a) (qpu-low-rewrite-cols-by-renames a rename-map))))
		expr)))

(define qpu-low-rewrite-fields-by-renames (lambda (fields rename-map)
	(map (coalesceNil fields '()) (lambda (pair) (match pair
		'(name expr) (list name (qpu-low-rewrite-cols-by-renames expr rename-map))
		pair)))))

(define qpu-low-join-key-left-expr-map (lambda (join-pred rhs-alias)
	(reduce (qpu-and-conjuncts join-pred) (lambda (acc c) (match c
		'(op lhs rhs)
		(if (qpu-low-equality-op? op)
			(begin
				(define li (qpu-low-col-ref-info lhs))
				(define ri (qpu-low-col-ref-info rhs))
				(if (and (not (nil? li)) (equal? (nth li 0) rhs-alias))
					(merge acc (list (list (nth li 1) rhs)))
					(if (and (not (nil? ri)) (equal? (nth ri 0) rhs-alias))
						(merge acc (list (list (nth ri 1) lhs)))
						acc)))
			acc)
		acc)) '())))

(define qpu-low-key-map-lookup (lambda (key-map col)
	(reduce (coalesceNil key-map '()) (lambda (acc pair) (match pair
		'(k v) (if (and (nil? acc) (equal? k col)) v acc)
		acc)) nil)))

(define qpu-low-rewrite-left-join-key-expr (lambda (expr key-map rhs-alias)
	(match expr
		'((symbol get_column) tv ti col ci)
		(if (equal? tv rhs-alias)
			(coalesce (qpu-low-key-map-lookup key-map col) expr)
			expr)
		'((quote get_column) tv ti col ci)
		(if (equal? tv rhs-alias)
			(coalesce (qpu-low-key-map-lookup key-map col) expr)
			expr)
		(cons head args)
		(cons head (map (coalesceNil args '())
			(lambda (a) (qpu-low-rewrite-left-join-key-expr a key-map rhs-alias))))
		expr)))

(define qpu-low-rewrite-left-join-key-fields (lambda (fields join-pred rhs-alias)
	(begin
		(define key-map (qpu-low-join-key-left-expr-map join-pred rhs-alias))
		(map (coalesceNil fields '()) (lambda (pair) (match pair
			'(name expr) (list name
				(qpu-low-rewrite-left-join-key-expr expr key-map rhs-alias))
			pair))))))

/* ==================== Lowering ==================== */

/* qpu-lower-to-tuple — top-level operator dispatch. */
(define qpu-lower-to-tuple (lambda (node)
	(match (qpir-kind node)
		(quote qpir-leaf)    (qpir-leaf-7tuple node)
		(quote qpir-select)  (qpu-low-select node)
		(quote qpir-map)     (qpu-low-map node)
		(quote qpir-groupby) (qpu-low-groupby node)
		(quote qpir-join)    (qpu-low-join node)
		(quote qpir-dep-join)
		(error "lower_to_scans_pass: qpir-dep-join remains in tree — unnest_pass must run first")
		(quote qpir-scan)
		(error "lower_to_scans_pass: qpir-scan not yet supported (qpir-leaf is the current leaf type)")
		(quote qpir-window)
		(error "lower_to_scans_pass: qpir-window not yet supported (phase 2+)")
		(quote qpir-union)
		(error "lower_to_scans_pass: qpir-union not yet supported (phase 2+)")
		(quote qpir-iterate)
		(error "lower_to_scans_pass: qpir-iterate not yet supported (phase 2+)")
		(error (concat "lower_to_scans_pass: unknown qpir kind "
			(string (qpir-kind node)))))))

(define qpu-low-select (lambda (node) (begin
	(define child-tuple (qpu-lower-to-tuple (qpir-select-child node)))
	/* Apply any sq_X.field rewrites from inline-flat scalars below us.
	The rewrites are registered in qpu-low-sq-rewrites via
	qpu-low-join-inline-scalar. We only apply them to the predicate
	when the child-tuple's tables list actually contains the registered
	rhs-alias — otherwise the rewrite belongs to a different scope (a
	nested scalar's inner WHERE, processed by its own qpu-low-select). */
	(define new-pred (qpu-low-sq-rewrites-apply-expr-scoped
		(qpir-select-predicate node)
		(qpp-tuple-tables child-tuple)))
	(define scalar-helper-source-has-aggregate? (lambda (src)
		(if (not (qpp-tuple? src)) false
			(or
				(> (count (coalesceNil (qpp-tuple-group src) '())) 0)
				(not (nil? (qpp-tuple-having src)))
				(reduce (qpp-fields-to-pairs (coalesceNil (qpp-tuple-fields src) '()))
					(lambda (acc pair) (match pair
						'(_ expr) (or acc (qpl-expr-has-aggregate? expr))
						acc)) false)))))
	(define scalar-helper-aliases
		(filter (map (coalesceNil (qpp-tuple-tables child-tuple) '()) (lambda (td)
			(match td
				'(a _ src io _) (if (and io (string? a) (>= (strlen a) 3)
					(equal? (substr a 0 3) "sq_")
					(not (scalar-helper-source-has-aggregate? src))) a nil)
				nil)))
			(lambda (a) (not (nil? a)))))
	(define scalar-helper-all-aliases
		(filter (map (coalesceNil (qpp-tuple-tables child-tuple) '()) (lambda (td)
			(match td
				'(a _ _ io _) (if (and io (string? a) (>= (strlen a) 3)
					(equal? (substr a 0 3) "sq_")) a nil)
				nil)))
			(lambda (a) (not (nil? a)))))
	(define expr-has-coalesce? (lambda (expr) (match expr
		'((symbol coalesce) _ _) true
		'((quote coalesce) _ _) true
		'((symbol coalesceNil) _ _) true
		'((quote coalesceNil) _ _) true
		(cons head args) (or (expr-has-coalesce? head)
			(reduce (coalesceNil args '()) (lambda (acc a)
				(or acc (expr-has-coalesce? a))) false))
		false)))
	(define expr-refs-alias? (lambda (expr alias)
		(reduce (qpir-expr-column-refs expr) (lambda (acc ref)
			(or acc (equal? (nth ref 0) alias))) false)))
	(define qpu-low-head-is? (lambda (head sym)
		(or (equal? head sym)
			(equal? head (symbol sym))
			(equal? head (list (quote quote) sym)))))
	(define coalesced-count-helper-ref (lambda (expr) (match expr
		'(head alias_ _ col _)
		(if (and (qpu-low-head-is? head (quote get_column))
			(equal? col "value")
			(has? scalar-helper-all-aliases alias_))
			alias_
			nil)
		nil)))
	(define coalesced-count-helper-target (lambda (expr) (match expr
		'(op lhs rhs)
		(if (and (qpu-low-head-is? op (quote >)) (equal? rhs 0))
			(match lhs
				'(cop inner default)
				(if (and (qpu-low-head-is? cop (quote coalesce)) (equal? default 0))
					(coalesced-count-helper-ref inner)
					nil)
				nil)
			nil)
		nil)))
	(define scalar-push-target (lambda (part)
		(coalesce
			(coalesced-count-helper-target part)
			(if (expr-has-coalesce? part) nil
				(reduce scalar-helper-aliases (lambda (found alias)
					(if (not (nil? found)) found
						(if (expr-refs-alias? part alias) alias nil))) nil)))))
	(define attach-predicate-to-table (lambda (tables target part)
		(map (coalesceNil tables '()) (lambda (td) (match td
			'(a s t io je)
			(if (equal? a target)
				(begin
					(define join-part (if (equal? (coalesced-count-helper-target part) target)
						nil
						part))
					(list a s t false (qpu-low-and-cond je join-part)))
				td)
			td)))))
	(define child-fields-pairs (qpp-fields-to-pairs (qpp-tuple-fields child-tuple)))
	(define child-has-agg-fields
		(reduce child-fields-pairs (lambda (acc pair) (match pair
			'(_ expr) (or acc (qpl-expr-has-aggregate? expr))
			acc)) false))
	(define strip-if-aggregate-select (lambda (tuple)
		(if child-has-agg-fields
			(qpu-low-strip-nested-scan-tags tuple)
			tuple)))
	(define scalar-domain-tuple
		(and
			(qpu-low-fields-has-name child-fields-pairs "value")
			(equal? (count child-fields-pairs) 1)
			(equal? (count (coalesceNil (qpp-tuple-group child-tuple) '())) 0)
			(nil? (qpp-tuple-having child-tuple))))
	(define nullify-count-target (coalesced-count-helper-target new-pred))
	(if (and scalar-domain-tuple (not (nil? nullify-count-target)))
		(strip-if-aggregate-select
			(qpp-rebuild-tuple
				(qpp-tuple-schema child-tuple)
				(qpp-tuple-tables child-tuple)
				(map child-fields-pairs (lambda (pair) (match pair
					'("value" expr) (list "value" (list (quote if) new-pred expr nil))
					pair)))
				(qpp-tuple-condition child-tuple)
				(qpp-tuple-group child-tuple)
				(qpp-tuple-having child-tuple)
				(qpp-tuple-order child-tuple)
				(qpp-tuple-limit child-tuple)
				(qpp-tuple-offset child-tuple)))
		(begin
			(define pushed (reduce (qpu-and-conjuncts new-pred) (lambda (acc part)
				(begin
					(define target (scalar-push-target part))
					(if (nil? target)
						(list (nth acc 0) (merge (nth acc 1) (list part)))
						(list (attach-predicate-to-table (nth acc 0) target part)
							(nth acc 1)))))
				(list (qpp-tuple-tables child-tuple) '())))
			(define pushed-tables (nth pushed 0))
			(define remaining-pred
				(reduce (nth pushed 1) (lambda (acc part)
					(qpu-low-and-cond acc part)) true))
			(define new-cond (qpu-low-and-cond
				(qpp-tuple-condition child-tuple)
				remaining-pred))
			(strip-if-aggregate-select
				(qpp-rebuild-tuple
					(qpp-tuple-schema child-tuple)
					pushed-tables
					(qpp-tuple-fields child-tuple)
					new-cond
					(qpp-tuple-group child-tuple)
					(qpp-tuple-having child-tuple)
					(qpp-tuple-order child-tuple)
					(qpp-tuple-limit child-tuple)
					(qpp-tuple-offset child-tuple))))))))

/* qpu-low-map-counter — counter for synthesized derived-wrapper aliases when
the map's child has aggregation that must be materialized before the map can
project. */
(define qpu-low-map-counter (newsession))
(qpu-low-map-counter "n" 0)
(define qpu-low-fresh-map-wrap (lambda () (begin
	(qpu-low-map-counter "n" (+ (qpu-low-map-counter "n") 1))
	(concat "__map_wrap_" (string (qpu-low-map-counter "n"))))))

/* qpu-low-fields-have-aggregate? — walk fields, return true if any
projection expression contains an (aggregate …) subexpr or a (window_func …)
subexpr. Used by qpu-low-map to detect "child has aggregation that must
materialize before map's projections". Fields format may be flat (parser)
or pairs (pipeline). */
(define qpu-low-expr-has-aggregate? (lambda (expr)
	(match expr
		'((symbol aggregate)   _ _ _) true
		'((quote aggregate)    _ _ _) true
		'((symbol window_func) _ _ _) true
		'((quote window_func)  _ _ _) true
		(cons head args) (reduce (coalesceNil args '()) (lambda (acc a)
			(or acc (qpu-low-expr-has-aggregate? a))) false)
		false)))

(define qpu-low-fields-have-aggregate? (lambda (fields)
	(reduce (qpp-fields-to-pairs fields) (lambda (acc pair) (match pair
		'(name expr) (or acc (qpu-low-expr-has-aggregate? expr))
		acc)) false)))

(define qpu-low-map (lambda (node) (begin
	(define child-tuple (qpu-lower-to-tuple (qpir-map-child node)))
	/* Apply scoped sq_X.field rewrites to map projections — sq_X tables may
	have been added by inline-flat below; map projections placed by lift
	may reference sq_X.value that needs the actual inner expr. */
	(define raw-projections (qpir-map-projections node))
	(define projections
		(map (coalesceNil raw-projections '()) (lambda (pair) (match pair
			'(n e) (list n (qpu-low-sq-rewrites-apply-expr-scoped e
				(qpp-tuple-tables child-tuple)))
			pair))))
	(define child-group (qpp-tuple-group child-tuple))
	(define child-has-group (and (not (nil? child-group)) (> (count child-group) 0)))
	/* Also wrap when child has aggregates in fields (e.g. static-group from
	qpir-groupby with empty keys lowers to group='() + N agg projections).
	Without this, qpu-low-map's blind field-replace loses the aggregate
	computations and the map's expression refs can't resolve. */
	(define child-explicit-group (and (not (nil? child-group))
		(equal? (count child-group) 0)
		(qpu-low-fields-have-aggregate? (qpp-tuple-fields child-tuple))))
	(define needs-wrap (or child-has-group child-explicit-group))
	(if needs-wrap
		/* Child has aggregation (qpir-groupby below): the GROUP BY must
		materialize BEFORE map's projections run, because map's expressions
		reference aggregate output columns. Wrap child as a derived table;
		the map's projections become the outer tuple's fields with refs
		resolved against the derived alias.

		The map's projections use (get_column nil false NAME false)
		placeholders for aggregate-output column refs (FAQ §35 canonical
		names). qpp-resolve-tuple-scoped (run AFTER lower if schemas
		provided) would qualify them; otherwise the nil-tv refs match the
		derived's schema via scope-fallback. */
		(begin
			(define wrap-alias (qpu-low-fresh-map-wrap))
			(define schema (qpp-tuple-schema child-tuple))
			(qpp-rebuild-tuple
				schema
				(list (list wrap-alias schema child-tuple false nil))
				(qpu-low-rewrite-map-projections projections wrap-alias)
				true
				nil nil nil nil nil))
		/* No GROUP BY: standard replace-fields path. */
		(qpp-rebuild-tuple
			(qpp-tuple-schema child-tuple)
			(qpp-tuple-tables child-tuple)
			projections
			(qpp-tuple-condition child-tuple)
			(qpp-tuple-group child-tuple)
			(qpp-tuple-having child-tuple)
			(qpp-tuple-order child-tuple)
			(qpp-tuple-limit child-tuple)
			(qpp-tuple-offset child-tuple))))))

/* qpu-low-rewrite-map-projections — qualify nil-tv refs in map projections
to the wrap-alias when wrapping the groupby child as derived. Without this,
the legacy resolver doesn't know the derived produces those names. */
(define qpu-low-rewrite-map-projections (lambda (projections wrap-alias)
	(map (coalesceNil projections '()) (lambda (pair) (match pair
		'(name expr) (list name (qpu-low-qualify-nil-refs expr wrap-alias))
		pair)))))

(define qpu-low-qualify-nil-refs (lambda (expr wrap-alias)
	(match expr
		'((symbol get_column) tv ti col ci)
		(if (nil? tv)
			(list (quote get_column) wrap-alias false col false)
			expr)
		'((quote get_column) tv ti col ci)
		(if (nil? tv)
			(list (quote get_column) wrap-alias false col false)
			expr)
		(cons head args)
		(cons head (map (coalesceNil args '())
			(lambda (a) (qpu-low-qualify-nil-refs a wrap-alias))))
		expr)))

(define qpu-low-groupby (lambda (node) (begin
	(define child-tuple
		(qpu-low-strip-nested-scan-tags
			(qpu-lower-to-tuple (qpir-groupby-child node))))
	/* Combine child's existing fields (the projected base columns) with the
	group-by key projections and the aggregate projections. The resulting
	7-tuple's fields list is what the GROUP BY query exposes. */
	(define agg-projections (qpir-groupby-aggs node))
	(define key-projections (nth
		(reduce (qpir-groupby-keys node) (lambda (acc key-expr)
			(begin
				(define raw-proj (qpu-low-key-projection key-expr))
				(define raw-name (nth raw-proj 0))
				(define unique-name (qpu-low-unique-projection-name raw-name
					(merge (nth acc 0) agg-projections)))
				(list
					(merge (nth acc 0) (list (list unique-name (nth raw-proj 1))))
					true)))
			(list '() true))
		0))
	(define new-fields (merge key-projections agg-projections))
	(qpp-rebuild-tuple
		(qpp-tuple-schema child-tuple)
		(qpp-tuple-tables child-tuple)
		new-fields
		(qpp-tuple-condition child-tuple)
		(qpir-groupby-keys node)
		(qpir-groupby-having node)
		(list)
		nil
		nil))))

/* qpu-low-join — convert a qpir-join into a merged 7-tuple. The right side
becomes part of the outer 7-tuple's tables. If the join has a rhs-alias
(which marks scalar-subquery-derived right sides per qpir-dep-join's
introduction), the right is wrapped as a derived table aliased rhs-alias
and column refs to the right's underlying tables are retargeted. */
(define qpu-low-join (lambda (node) (begin
	(define left-tuple (qpu-lower-to-tuple (qpir-join-left node)))
	(define right-tuple (qpu-lower-to-tuple (qpir-join-right node)))
	(define rhs-alias (qpir-join-rhs-alias node))
	(define join-pred (qpir-join-predicate node))
	(define jtype (qpir-join-type node))
	(if (nil? rhs-alias)
		(qpu-low-join-merge-tables left-tuple right-tuple join-pred)
		/* Inline-flat for correlated scalar dep-joins per FAQ §20+§43.
		The derived-wrap fallback handles cases inline-flat doesn't fit
		(multi-table inner, aggregates, complex right tree). */
		(begin
			(define left-aliases (map (qpp-tuple-tables left-tuple) (lambda (t)
				(if (or (nil? t) (< (count t) 1)) nil (nth t 0)))))
			/* Gate on OUTER shape too: inline-flat is unsafe when outer has
			GROUP BY or aggregate-bearing fields. The inlined inner table
			flat-joins to outer; aggregates like COUNT(*) then count joined
			rows instead of outer rows. Wrap-derived keeps the scalar self-
			contained, which legacy aggregates correctly. */
			(define outer-has-group
				(> (count (coalesceNil (qpp-tuple-group left-tuple) '())) 0))
			(define outer-has-agg-fields
				(reduce (qpp-fields-to-pairs
					(coalesceNil (qpp-tuple-fields left-tuple) '()))
					(lambda (acc pair) (match pair
						'(_ e) (or acc (qpl-expr-has-aggregate? e))
						acc)) false))
			(define right-has-explicit-limit-one
				(equal? (qpp-tuple-limit right-tuple) 1))
			(if (and (not outer-has-group)
				(or (not outer-has-agg-fields) right-has-explicit-limit-one)
				(qpu-low-inline-scalar-eligible? right-tuple join-pred
					rhs-alias jtype left-aliases))
				(qpu-low-join-inline-scalar left-tuple right-tuple join-pred
					rhs-alias jtype)
				(qpu-low-join-wrap-derived left-tuple right-tuple join-pred
					rhs-alias jtype)))))))

/* qpu-low-join-merge-tables — for a join WITHOUT rhs-alias: append the
right's tables into the left's tables list and AND conditions/predicate. */
(define qpu-low-join-merge-tables (lambda (left-tuple right-tuple join-pred)
	(qpp-rebuild-tuple
		(qpp-tuple-schema left-tuple)
		(merge (qpp-tuple-tables left-tuple) (qpp-tuple-tables right-tuple))
		(qpp-tuple-fields left-tuple)
		(qpu-low-and-cond
			(qpu-low-and-cond
				(qpp-tuple-condition left-tuple)
				(qpp-tuple-condition right-tuple))
			join-pred)
		(qpp-tuple-group left-tuple)
		(qpp-tuple-having left-tuple)
		(qpp-tuple-order left-tuple)
		(qpp-tuple-limit left-tuple)
		(qpp-tuple-offset left-tuple))))

/* qpu-low-tag-inner-once-limit — for the simplest scalar-context derived
shape, replace the inner sub-tuple's single base table tname with a
`(scan-tagged-table ...)` carrying once_limit=2 per FAQ §20 / once-limit-
rework. This makes legacy build_scan emit a per-scan once-promise that
errors on the 2nd row, enforcing SQL scalar cardinality.

Conservative gate (avoid the "map expects a list, got X" failures observed
in earlier broader attempts):
- exactly 1 table entry in the sub
- tname is a string (not nested derived 7-tuple, not already-tagged)
- sub has no LIMIT (LIMIT 1 is its own contract handled elsewhere)
- sub has no aggregate-bearing fields (aggregation is per-group ≤1 row
anyway, and helper scans for aggregates break the simple tag) */
(define qpu-low-tag-inner-once-limit (lambda (sub-tuple)
	(begin
		(define tbls (coalesceNil (qpp-tuple-tables sub-tuple) '()))
		(define has-lim (not (nil? (qpp-tuple-limit sub-tuple))))
		(define flds (qpp-fields-to-pairs
			(coalesceNil (qpp-tuple-fields sub-tuple) '())))
		(define has-agg (reduce flds (lambda (acc pair) (match pair
			'(_ e) (or acc (qpl-expr-has-aggregate? e)
				/* count_distinct is parser-emitted and gets rewritten to
				(aggregate ...) only inside untangle_query — by lower
				time we still see it as (count_distinct expr). Treat as
				aggregate so we don't tag its inner table. */
				(qpu-low-tag-has-count-distinct? e))
			acc)) false))
		(if (or has-lim has-agg
			(not (equal? (count flds) 1))
			(not (equal? (count tbls) 1)))
			sub-tuple
			(begin
				(define td (nth tbls 0))
				(match td
					'(td-alias td-schema td-tname td-io td-je)
					(if (or (qpp-tuple? td-tname)
						(and (list? td-tname) (> (count td-tname) 0)
							(or (equal? (car td-tname) (quote scan-tagged-table))
								(equal? (car td-tname) (symbol scan-tagged-table)))))
						sub-tuple
						(begin
							(define tagged (make_scan_tagged_table td-tname '() 2 nil 0 2))
							(define new-td (list td-alias td-schema tagged td-io td-je))
							(define new-tbls (cons new-td (cdr tbls)))
							(qpp-rebuild-tuple
								(qpp-tuple-schema sub-tuple)
								new-tbls
								(qpp-tuple-fields sub-tuple)
								(qpp-tuple-condition sub-tuple)
								(qpp-tuple-group sub-tuple)
								(qpp-tuple-having sub-tuple)
								(qpp-tuple-order sub-tuple)
								(qpp-tuple-limit sub-tuple)
								(qpp-tuple-offset sub-tuple))))
					sub-tuple))))))

(define qpu-low-tag-expr-has-aggregate? (lambda (expr) (match expr
	'((symbol aggregate) . _) true
	'((quote aggregate) . _) true
	(cons head args) (or (qpu-low-tag-expr-has-aggregate? head)
		(reduce (coalesceNil args '()) (lambda (acc a)
			(or acc (qpu-low-tag-expr-has-aggregate? a))) false))
	false)))

/* ==================== Inline-flat scalar lowering (FAQ §43) ==================== */

/* qpu-low-col-ref-info — extract (tv col) from a get_column expr, or nil. */
(define qpu-low-col-ref-info (lambda (expr) (match expr
	'((symbol get_column) tv _ col _) (list tv col)
	'((quote get_column) tv _ col _) (list tv col)
	nil)))

/* qpu-low-corr-from-pred — given join-pred + inner alias + outer aliases,
return list of inner-side column names paired with outer cols via equality
conjuncts. Order preserves first-seen. */
(define qpu-low-corr-from-pred (lambda (join-pred inner-alias outer-aliases)
	(begin
		(define seen (newsession))
		(seen "cols" '())
		(define handle-eq (lambda (lhs rhs)
			(begin
				(define li (qpu-low-col-ref-info lhs))
				(define ri (qpu-low-col-ref-info rhs))
				(define inner-col
					(if (and (not (nil? li)) (equal? (nth li 0) inner-alias)
						(not (nil? ri)) (has? outer-aliases (nth ri 0)))
						(nth li 1)
						(if (and (not (nil? ri)) (equal? (nth ri 0) inner-alias)
							(not (nil? li)) (has? outer-aliases (nth li 0)))
							(nth ri 1)
							nil)))
				(if (and (not (nil? inner-col))
					(not (has? (seen "cols") inner-col)))
					(seen "cols" (merge (seen "cols") (list inner-col)))
					nil))))
		(reduce (qpu-and-conjuncts join-pred) (lambda (acc c)
			(begin
				(match c
					'((symbol equal??) lhs rhs) (handle-eq lhs rhs)
					'((quote equal??)  lhs rhs) (handle-eq lhs rhs)
					'((symbol =)       lhs rhs) (handle-eq lhs rhs)
					'((quote =)        lhs rhs) (handle-eq lhs rhs)
					nil)
				acc)) nil)
		(seen "cols"))))

/* qpu-low-inline-scalar-eligible? — true when the scalar dep-join's right
side can be inlined as flat (instead of derived-wrap). Conservative gate
matching qpu-low-tag-inner-once-limit:
- join-type=left
- rhs-alias starts with sq_
- right has 1 table entry, base table (not derived/tagged)
- right has 1 field (the scalar value)
- right has no group/having (aggregates use derived path)
- right's field has no aggregate / count_distinct
- join-pred is NOT trivially-true (correlated case — uncorrelated keeps
Phase 1 path which is simpler)
- extracted correlation cols match the inner table's alias */
(define qpu-low-inline-scalar-eligible? (lambda (right-tuple join-pred rhs-alias join-type left-aliases)
	(begin
		(define tbls (coalesceNil (qpp-tuple-tables right-tuple) '()))
		(define flds (qpp-fields-to-pairs
			(coalesceNil (qpp-tuple-fields right-tuple) '())))
		(define grp (coalesceNil (qpp-tuple-group right-tuple) '()))
		(define hav (qpp-tuple-having right-tuple))
		(if (not (and
			(equal? join-type (quote left))
			(string? rhs-alias)
			(>= (strlen rhs-alias) 3)
			(equal? (substr rhs-alias 0 3) "sq_")
			(equal? (count tbls) 1)
			(equal? (count flds) 1)
			(equal? (count grp) 0)
			(nil? hav)
			/* correlated only — uncorrelated has dedicated tag path */
			(not (or (nil? join-pred) (equal? join-pred true)
				(equal? join-pred (quote true))))))
			false
			(begin
				(define td (nth tbls 0))
				(match td
					'(td-alias _ td-tname _ _)
					(if (or (qpp-tuple? td-tname)
						(and (list? td-tname) (> (count td-tname) 0)
							(or (equal? (car td-tname) (quote scan-tagged-table))
								(equal? (car td-tname) (symbol scan-tagged-table)))))
						false
						(begin
							(define fpair (nth flds 0))
							(define fexpr (nth fpair 1))
							(if (or (qpl-expr-has-aggregate? fexpr)
								(qpu-low-tag-has-count-distinct? fexpr))
								false
								(begin
									/* Must extract at least 1 correlation column */
									(define cc (qpu-low-corr-from-pred join-pred td-alias left-aliases))
									(> (count cc) 0)))))
					false)))))) */
false))))))

/* qpu-low-join-inline-scalar — produce a flat-join outer tuple instead of
wrapping right as derived. Inline the inner table with scan-tagged-table
carrying correlation cols at front of sortcols and partition_cols=N.
Outer field refs to sq_X.value get rewritten to inner-aliased col directly.

Aliasing: the inner table is REALIASED to rhs-alias (sq_X) so multiple
scalars over the same base table (e.g. two identical correlated scalars
in the same SELECT list) don't collide on the inner natural alias. All
refs in join-pred, inner-where, and field-expr get retargeted from
inner-natural-alias → rhs-alias.

Layout:
- outer's tables: existing + (rhs-alias inner_schema tagged-tname true joinExpr)
- joinExpr = retargeted join-pred AND retargeted inner-where
- outer fields/cond: refs (rhs-alias . field-name) → retargeted field-expr
*/
(define qpu-low-join-inline-scalar (lambda (left-tuple right-tuple join-pred rhs-alias join-type)
	(begin
		(define tbls (qpp-tuple-tables right-tuple))
		(define td (nth tbls 0))
		(define inner-natural-alias (nth td 0))
		(define inner-schema (nth td 1))
		(define inner-base (nth td 2))
		(define inner-where (qpp-tuple-condition right-tuple))
		(define inner-order (coalesceNil (qpp-tuple-order right-tuple) '()))
		(define inner-offset (qpp-tuple-offset right-tuple))
		(define sub-lim (qpp-tuple-limit right-tuple))
		(define fpair (nth (qpp-fields-to-pairs (qpp-tuple-fields right-tuple)) 0))
		(define field-name (nth fpair 0))
		(define field-expr-raw (nth fpair 1))
		(define left-aliases (map (qpp-tuple-tables left-tuple) (lambda (t)
			(if (or (nil? t) (< (count t) 1)) nil (nth t 0)))))
		(define corr-cols (qpu-low-corr-from-pred join-pred inner-natural-alias left-aliases))
		/* Retarget all refs to inner-natural-alias → rhs-alias */
		(define retarget-aliases (list inner-natural-alias))
		(define field-expr (qpu-low-rewrite-refs field-expr-raw retarget-aliases rhs-alias))
		(define inner-where-clean
			(if (or (nil? inner-where) (equal? inner-where true)
				(equal? inner-where (quote true))) nil inner-where))
		(define inner-where-retargeted
			(if (nil? inner-where-clean) nil
				(qpu-low-rewrite-refs inner-where-clean retarget-aliases rhs-alias)))
		(define join-pred-retargeted (qpu-low-rewrite-refs
			join-pred retarget-aliases rhs-alias))
		(define corr-order
			(map corr-cols (lambda (c) (list
				(list (quote get_column) rhs-alias false c false)
				(quote <)))))
		(define inner-order-retargeted
			(map inner-order (lambda (o) (match o
				'(c d) (list (qpu-low-rewrite-refs c retarget-aliases rhs-alias) d)
				o))))
		(define stag-order (merge corr-order inner-order-retargeted))
		/* Limit + once_limit per once-limit-rework. */
		(define stag-limit
			(if (nil? sub-lim) 2
				(if (equal? sub-lim 1) 1 sub-lim)))
		(define stag-once-limit
			(if (nil? sub-lim) 2
				(if (equal? sub-lim 1) 1 0)))
		(define tagged-tname (make_scan_tagged_table
			inner-base stag-order stag-limit inner-offset
			(count corr-cols) stag-once-limit))
		(define combined-je (qpu-low-and-cond join-pred-retargeted inner-where-retargeted))
		(define new-entry
			(list rhs-alias inner-schema tagged-tname true combined-je))
		/* Register the (rhs-alias . field-name) → field-expr rewrite so any
		OUTER wrapper (qpir-select, qpir-map, qpir-groupby) above us picks
		it up when applying its own predicates/projections. Lift placed
		sq_X.field refs in the outer's WHERE/SELECT — those scopes are
		processed at a different lowering step than ours. */
		(qpu-low-sq-rewrites-add rhs-alias field-name field-expr)
		/* (rewrite map populated for any future wrapper that needs it —
		currently only used by the local rewrite below; the qpu-low-select
		integration is deferred since it caused esv test regressions —
		trades 1 esv win for 4 esv losses.) */
		(define rewritten-fields
			(qpu-low-replace-sq-field (qpp-tuple-fields left-tuple)
				rhs-alias field-name field-expr))
		(define rewritten-cond
			(qpu-low-replace-sq-field-expr (qpp-tuple-condition left-tuple)
				rhs-alias field-name field-expr))
		(define rewritten-having
			(if (nil? (qpp-tuple-having left-tuple)) nil
				(qpu-low-replace-sq-field-expr (qpp-tuple-having left-tuple)
					rhs-alias field-name field-expr)))
		(qpp-rebuild-tuple
			(qpp-tuple-schema left-tuple)
			(merge (qpp-tuple-tables left-tuple) (list new-entry))
			rewritten-fields
			rewritten-cond
			(qpp-tuple-group left-tuple)
			rewritten-having
			(qpp-tuple-order left-tuple)
			(qpp-tuple-limit left-tuple)
			(qpp-tuple-offset left-tuple)))))

(define qpu-low-replace-sq-field-expr (lambda (expr rhs-alias field-name target-expr)
	(match expr
		'((symbol get_column) tv _ col _)
		(if (and (equal? tv rhs-alias) (equal? col field-name))
			target-expr expr)
		'((quote get_column) tv _ col _)
		(if (and (equal? tv rhs-alias) (equal? col field-name))
			target-expr expr)
		(cons head args)
		(cons (qpu-low-replace-sq-field-expr head rhs-alias field-name target-expr)
			(map (coalesceNil args '()) (lambda (a)
				(qpu-low-replace-sq-field-expr a rhs-alias field-name target-expr))))
		expr)))

(define qpu-low-replace-sq-field (lambda (fields rhs-alias field-name target-expr)
	(map (coalesceNil fields '()) (lambda (pair) (match pair
		'(fn fe) (list fn (qpu-low-replace-sq-field-expr fe rhs-alias field-name target-expr))
		pair)))))

/* qpu-low-sq-rewrites — session map of pending sq_X.field → target-expr
substitutions populated by qpu-low-join-inline-scalar. Consumed by
qpu-low-select / qpu-low-map / qpu-low-groupby when they wrap the inline-
flat result; the outer's WHERE/projections/keys may contain refs to
sq_X.value (from lift's marker substitution) that the inline-flat path
must rewrite to the inner column expr.

Cleared at the start of each lower_to_scans_pass invocation so cross-
query state doesn't leak. */
(define qpu-low-sq-rewrites (newsession))
(qpu-low-sq-rewrites "list" '())

(define qpu-low-sq-rewrites-clear (lambda ()
	(qpu-low-sq-rewrites "list" '())))

(define qpu-low-sq-rewrites-add (lambda (rhs-alias field-name target-expr)
	(qpu-low-sq-rewrites "list"
		(merge (coalesceNil (qpu-low-sq-rewrites "list") '())
			(list (list rhs-alias field-name target-expr))))))

(define qpu-low-sq-rewrites-apply-expr (lambda (expr)
	(reduce (coalesceNil (qpu-low-sq-rewrites "list") '())
		(lambda (acc entry) (match entry
			'(ra fn tgt) (qpu-low-replace-sq-field-expr acc ra fn tgt)
			acc))
		expr)))

(define qpu-low-sq-rewrites-apply-fields (lambda (fields)
	(reduce (coalesceNil (qpu-low-sq-rewrites "list") '())
		(lambda (acc entry) (match entry
			'(ra fn tgt) (qpu-low-replace-sq-field acc ra fn tgt)
			acc))
		fields)))

(define qpu-low-sq-rewrites-apply-order (lambda (order)
	(map (coalesceNil order '()) (lambda (o) (match o
		'(c d) (list (qpu-low-sq-rewrites-apply-expr c) d)
		o)))))

/* qpu-low-sq-rewrites-apply-expr-scoped — apply ONLY those rewrites whose
rhs-alias appears in `tables`. This scopes the rewrite to the level where
the inlined sq_X table is actually visible, preventing a nested scalar's
rewrite from leaking into a sibling/outer scope's predicate. */
(define qpu-low-sq-rewrites-apply-expr-scoped (lambda (expr tables) (begin
	(define table-aliases (map (coalesceNil tables '()) (lambda (td)
		(if (or (nil? td) (< (count td) 1)) nil (nth td 0)))))
	(reduce (coalesceNil (qpu-low-sq-rewrites "list") '())
		(lambda (acc entry) (match entry
			'(ra fn tgt)
			(if (has? table-aliases ra)
				(qpu-low-replace-sq-field-expr acc ra fn tgt)
				acc)
			acc))
		expr))))

(define qpu-low-tag-has-count-distinct? (lambda (expr) (match expr
	(cons head args) (or
		(equal? head (quote count_distinct))
		(equal? head (symbol count_distinct))
		(qpu-low-tag-has-count-distinct? head)
		(reduce (coalesceNil args '()) (lambda (acc a)
			(or acc (qpu-low-tag-has-count-distinct? a))) false))
	false)))

(define qpu-low-field-expr-by-name (lambda (fields name)
	(reduce (coalesceNil fields '()) (lambda (acc pair) (match pair
		'(fn fe) (if (and (nil? acc) (equal? fn name)) fe acc)
		acc)) nil)))

(define qpu-low-order-only-alias? (lambda (order alias)
	(reduce (coalesceNil order '()) (lambda (acc oi)
		(if (and (list? oi) (> (count oi) 0))
			(begin
				(define expr (nth oi 0))
				(and acc (reduce (qpir-expr-column-refs expr) (lambda (ok ref)
					(if (and (list? ref) (> (count ref) 0))
						(and ok (equal? (nth ref 0) alias))
						ok)) true)))
			acc)) true)))

(define qpu-low-direct-key-projections (lambda (fields rename-map)
	(filter
		(map (coalesceNil rename-map '()) (lambda (entry)
			(if (and (list? entry) (equal? (count entry) 2)
				(list? (nth entry 0)) (equal? (count (nth entry 0)) 2))
				(begin
					(define refpair (nth entry 0))
					(define rtv (nth refpair 0))
					(define pname (nth entry 1))
					(define pexpr (qpu-low-field-expr-by-name fields pname))
					(define pinfo (qpu-low-col-ref-info pexpr))
					(if (and (not (nil? pinfo)) (equal? (nth pinfo 0) rtv))
						(list (nth pinfo 0) (nth pinfo 1) pexpr)
						nil))
				nil)))
		(lambda (x) (not (nil? x))))))

(define qpu-low-tag-correlated-ordered-scalar-limit (lambda (sub-tuple rename-map)
	(begin
		(define tbls (coalesceNil (qpp-tuple-tables sub-tuple) '()))
		(define flds (qpp-fields-to-pairs
			(coalesceNil (qpp-tuple-fields sub-tuple) '())))
		(define ord (coalesceNil (qpp-tuple-order sub-tuple) '()))
		(define lim (qpp-tuple-limit sub-tuple))
		(define off (qpp-tuple-offset sub-tuple))
		(define key-projs (qpu-low-direct-key-projections flds rename-map))
		(if (or
			(not (equal? lim 1))
			(not (nil? off))
			(equal? (count ord) 0)
			(equal? (count tbls) 0)
			(equal? (count key-projs) 0))
			sub-tuple
			(begin
				(define key-alias (nth (car key-projs) 0))
				(define all-key-same-alias
					(reduce key-projs (lambda (acc kp)
						(and acc (equal? (nth kp 0) key-alias))) true))
				(define first-td (car tbls))
				(match first-td
					'(td-alias td-schema td-tname td-io td-je)
					(if (or
						(not all-key-same-alias)
						(not (equal? td-alias key-alias))
						(not (qpu-low-order-only-alias? ord key-alias))
						(qpp-tuple? td-tname)
						(qpu-low-scan-tagged-table? td-tname))
						sub-tuple
						(begin
							(define partition-order
								(map key-projs (lambda (kp)
									(list (nth kp 2) (quote <)))))
							(define tagged (make_scan_tagged_table
								td-tname
								(merge partition-order ord)
								1
								nil
								(count partition-order)
								1))
							(qpp-rebuild-tuple
								(qpp-tuple-schema sub-tuple)
								(cons (list td-alias td-schema tagged td-io td-je)
									(cdr tbls))
								(qpp-tuple-fields sub-tuple)
								(qpp-tuple-condition sub-tuple)
								(qpp-tuple-group sub-tuple)
								(qpp-tuple-having sub-tuple)
								(qpp-tuple-order sub-tuple)
								(qpp-tuple-limit sub-tuple)
								(qpp-tuple-offset sub-tuple))))
					sub-tuple)))))))

/* qpu-low-join-wrap-derived — for a join WITH rhs-alias: wrap the right's
7-tuple as a derived-table entry aliased rhs-alias. Outer column references
to the right's underlying tables are retargeted to rhs-alias so they
resolve through the derived table's projection.

Per lib/sql-parser.scm tabledef: a derived table is
(alias schema sub-7-tuple isOuter joinExpr)

For qpir-join-type = left (FAQ §22 per-key-misses): isOuter=true and the
join predicate becomes the joinExpr (so per-key misses get NULL-extended,
not filtered out).

For qpir-join-type = inner: isOuter=false; predicate goes into the outer's
WHERE condition (existing behavior). */
(define qpu-low-join-wrap-derived (lambda (left-tuple right-tuple join-pred rhs-alias join-type)
	(begin
		(define right-source-aliases (map (qpp-tuple-tables right-tuple) (lambda (td)
			(if (or (nil? td) (< (count td) 1)) nil (nth td 0)))))
		/* Outer's tables (LEFT) — when an alias appears in BOTH outer and
		right, that alias name collides. Outer's existing field/cond refs to
		the colliding alias are LEGITIMATE outer references and must NOT be
		retargeted to rhs-alias. Only the join-pred (which carries the
		synthesized correlation against right-source tables) gets full
		retargeting. */
		(define left-source-aliases (map (qpp-tuple-tables left-tuple) (lambda (td)
			(if (or (nil? td) (< (count td) 1)) nil (nth td 0)))))
		(define right-only-aliases (filter right-source-aliases (lambda (a)
			(not (has? left-source-aliases a)))))
		/* Ensure the right-tuple exposes every column referenced by the join
		predicate against its underlying tables. Without this, retargeting
		`(t4 id)` to `(sq_N id)` produces a reference to a non-existent
		field of the derived sub-tuple. The rename-map records (tv col) →
		projected-name so we can rewrite the predicate with the correct
		(possibly synthesized) field name. */
		(define keys-result (qpu-low-ensure-join-key-fields
			right-tuple join-pred right-source-aliases))
		(define right-tuple-keys-raw (nth keys-result 0))
		(define key-rename-map (nth keys-result 1))
		(define is-left (equal? join-type (quote left)))
		(define right-join-pred (qpu-low-rewrite-cols-by-renames
			join-pred key-rename-map))
		(define right-tuple-keys-dedup
			(if is-left
				(qpp-rebuild-tuple
					(qpp-tuple-schema right-tuple-keys-raw)
					(qpp-tuple-tables right-tuple-keys-raw)
					(qpp-tuple-fields right-tuple-keys-raw)
					(qpu-low-remove-conjunct
						(qpp-tuple-condition right-tuple-keys-raw)
						right-join-pred)
					(qpp-tuple-group right-tuple-keys-raw)
					(qpp-tuple-having right-tuple-keys-raw)
					(qpp-tuple-order right-tuple-keys-raw)
					(qpp-tuple-limit right-tuple-keys-raw)
					(qpp-tuple-offset right-tuple-keys-raw))
				right-tuple-keys-raw))
		/* For UNCORRELATED sq_*-aliased scalar derived: tag the inner base
		table with scan-tagged-table once_limit=2 so multi-row inner scans
		error per FAQ §20. CORRELATED scalars: tried partition-tagging in
		tick 12 of session 2026-05-31 — regressed -30 tests. The flat
		derived path doesn't have enough scope to set partition_cols
		correctly without breaking sibling queries. Correlated remains on
		the existing derived-wrap path until a proper inline-flat refactor
		restructures the IR (FAQ §43 plan). */
		(define is-scalar-rhs
			(and (string? rhs-alias) (>= (strlen rhs-alias) 3)
				(equal? (substr rhs-alias 0 3) "sq_")))
		(define is-uncorrelated
			(or (nil? join-pred) (equal? join-pred true)
				(equal? join-pred (quote true))))
		(define right-tuple-keys
			(if (and is-scalar-rhs is-uncorrelated)
				(qpu-low-tag-inner-once-limit right-tuple-keys-dedup)
				(if (and is-scalar-rhs is-left (not is-uncorrelated))
					(qpu-low-tag-correlated-ordered-scalar-limit
						right-tuple-keys-dedup key-rename-map)
					right-tuple-keys-dedup)))
		(define local-rewritten-pred (qpu-low-rewrite-by-renames
			join-pred key-rename-map rhs-alias))
		(define raw-rewritten-fields (qpu-low-rewrite-projections
			(qpp-tuple-fields left-tuple) right-only-aliases rhs-alias))
		(define renamed-rewritten-fields (qpu-low-rewrite-fields-by-renames
			raw-rewritten-fields key-rename-map))
		(define rewritten-fields
			(if is-left
				(qpu-low-rewrite-left-join-key-fields
					renamed-rewritten-fields local-rewritten-pred rhs-alias)
				renamed-rewritten-fields))
		(define rewritten-cond (qpu-low-rewrite-refs
			(qpp-tuple-condition left-tuple) right-only-aliases rhs-alias))
		/* Rewrite join-pred using the rename-map so synthesized __kt_* names
		are referenced under rhs-alias. Refs not in rename-map use the
		regular rewrite (alias bump). */
		(define rewritten-pred (qpu-low-rewrite-refs
			local-rewritten-pred
			right-source-aliases rhs-alias))
		(define normalized-rewritten-pred
			(qpu-simplify-predicate
				(qpu-low-normalize-kt-refs-in-expr
					rewritten-pred
					(list rhs-alias)
					(qpp-fields-to-pairs (qpp-tuple-fields right-tuple-keys)))))
		(define derived-entry
			(if is-left
				/* LEFT join: derived table is isOuter=true with joinExpr = pred.
				Per-key misses get NULL-extended automatically by the scan
				infrastructure (FAQ §22 isOuter contract). */
				(list rhs-alias (qpp-tuple-schema right-tuple-keys)
					right-tuple-keys true normalized-rewritten-pred)
				/* INNER join: derived table is plain; predicate flows into WHERE. */
				(list rhs-alias (qpp-tuple-schema right-tuple-keys)
					right-tuple-keys false nil)))
		(define final-cond
			(if is-left
				/* LEFT: predicate is in joinExpr, NOT in WHERE (else inner-join semantics). */
				rewritten-cond
				(qpu-low-and-cond rewritten-cond normalized-rewritten-pred)))
		(define post-group-scalar?
			(and is-scalar-rhs is-uncorrelated (not is-left)
				(> (count (coalesceNil (qpp-tuple-group left-tuple) '())) 0)
				(not (nil? (qpp-tuple-having left-tuple)))))
		(if post-group-scalar?
			(begin
				(define group-alias (qpu-low-fresh-map-wrap))
				(define left-fields (qpp-fields-to-pairs (qpp-tuple-fields left-tuple)))
				(define field-name-for-expr (lambda (expr)
					(reduce left-fields (lambda (found pair) (match pair
						'(name fexpr) (if (or (not (nil? found)) (not (equal? expr fexpr)))
							found
							name)
						found)) nil)))
				(define rewrite-post-group-expr (lambda (expr) (begin
					(define exact-name (field-name-for-expr expr))
					(if (not (nil? exact-name))
						(list (quote get_column) group-alias false exact-name false)
						(match expr
							(cons head args)
							(cons head (map (coalesceNil args '()) rewrite-post-group-expr))
							expr)))))
				(define grouped-left (qpp-rebuild-tuple
					(qpp-tuple-schema left-tuple)
					(qpp-tuple-tables left-tuple)
					(qpp-tuple-fields left-tuple)
					(qpp-tuple-condition left-tuple)
					(qpp-tuple-group left-tuple)
					nil
					nil
					nil
					nil))
				(qpp-rebuild-tuple
					(qpp-tuple-schema left-tuple)
					(merge
						(list (list group-alias (qpp-tuple-schema left-tuple) grouped-left false nil))
						(list derived-entry))
					(map left-fields (lambda (pair) (match pair
						'(name _) (list name (list (quote get_column) group-alias false name false))
						pair)))
					(qpu-low-and-cond
						(rewrite-post-group-expr (qpp-tuple-having left-tuple))
						(if is-left true normalized-rewritten-pred))
					nil
					nil
					(qpp-tuple-order left-tuple)
					(qpp-tuple-limit left-tuple)
					(qpp-tuple-offset left-tuple)))
			(qpp-rebuild-tuple
				(qpp-tuple-schema left-tuple)
				(merge (qpp-tuple-tables left-tuple) (list derived-entry))
				rewritten-fields
				final-cond
				(qpp-tuple-group left-tuple)
				(qpp-tuple-having left-tuple)
				(qpp-tuple-order left-tuple)
				(qpp-tuple-limit left-tuple)
				(qpp-tuple-offset left-tuple)))))))

/* ==================== Public driver ==================== */

/* lower_to_scans_pass — the L4 transformation.
Takes a qpir tree (post-unnest, no dep-joins, F(root)=∅) and returns a
single 7-tuple compatible with the legacy build_queryplan_inner. The
caller then feeds this 7-tuple into the existing physical compiler for
scan/keytable/join emission. */
(define lower_to_scans_pass (lambda (qpir-tree) (begin
	(qpu-low-sq-rewrites-clear)
	(define lowered (qpu-lower-to-tuple qpir-tree))
	(if (not (qpp-tuple? lowered))
		(error "lower_to_scans_pass: lowering did not produce a 7-tuple")
		lowered))))
