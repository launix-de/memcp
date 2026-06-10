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
qpir-join inner (po.k = pi.k) rhs-alias=domain_scalar_payments_1
qpir-leaf po
qpir-groupby [pi.k] [(value SUM)] nil
qpir-leaf pi
→
7-tuple {tables: [po, (domain_scalar_payments_1 schema <group-7-tuple> false nil)],
fields: [(id po.id) (total domain_scalar_payments_1.value)],
condition: (po.k = domain_scalar_payments_1.k)}
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

(define qpu-low-has-conjunct? (lambda (cond target)
	(if (or (nil? cond) (equal? cond true) (equal? cond (quote true)))
		false
		(if (or (equal? cond target) (qpu-low-same-equality? cond target))
			true
			(match cond
				'((symbol and) a b)
				(or (qpu-low-has-conjunct? a target)
					(qpu-low-has-conjunct? b target))
				'((quote and) a b)
				(or (qpu-low-has-conjunct? a target)
					(qpu-low-has-conjunct? b target))
				false)))))

(define qpu-low-equality-conjunct? (lambda (expr)
	(match expr
		'(op _ _) (qpu-low-equality-op? op)
		false)))

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
				(if (qpu-low-scalar-helper-alias? to-alias)
					(qpu-low-kt-ref-source-col col)
					col)
				false)
			expr)
		'((quote get_column) tv ti col ci)
		(if (has? from-aliases tv)
			(list (quote get_column) to-alias false
				(if (qpu-low-scalar-helper-alias? to-alias)
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
		'(n e) (or acc (equal?? n name))
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

Without the cascade case, join-pred retargeting `(r src)` → `(domain_scalar_* src)`
produces a reference to a non-existent field — the helper sub-tuple doesn't
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

(define qpu-low-expr-has-ref-outside? (lambda (expr local-aliases)
	(match expr
		'((symbol get_column) tv _ _ _) (not (has? local-aliases tv))
		'((quote get_column) tv _ _ _) (not (has? local-aliases tv))
		(cons head args)
		(or (qpu-low-expr-has-ref-outside? head local-aliases)
			(reduce (coalesceNil args '()) (lambda (acc a)
				(or acc (qpu-low-expr-has-ref-outside? a local-aliases))) false))
		false)))

(define qpu-low-add-outer-hop-for-nonlocal-refs (lambda (expr local-aliases)
	(match expr
		'((symbol outer) _) (if (qpu-low-expr-has-ref-outside? expr local-aliases)
			(list (quote outer) expr)
			expr)
		'((quote outer) _) (if (qpu-low-expr-has-ref-outside? expr local-aliases)
			(list (quote outer) expr)
			expr)
		'((symbol get_column) tv _ _ _) (if (has? local-aliases tv)
			expr
			(match expr
				'(_ alias_ _ col_ _) (list (quote outer)
					(symbol (concat alias_ "." col_)))
				(list (quote outer) expr)))
		'((quote get_column) tv _ _ _) (if (has? local-aliases tv)
			expr
			(match expr
				'(_ alias_ _ col_ _) (list (quote outer)
					(symbol (concat alias_ "." col_)))
				(list (quote outer) expr)))
		(cons head args)
		(cons head (map (coalesceNil args '())
			(lambda (a) (qpu-low-add-outer-hop-for-nonlocal-refs a local-aliases))))
		expr)))

(define qpu-low-carry-left-tables-one-scope-deeper (lambda (tables local-aliases)
	(map (coalesceNil tables '()) (lambda (td) (match td
		'(a s t io je)
		(list a s t io
			(if (nil? je) nil
				(qpu-low-add-outer-hop-for-nonlocal-refs je local-aliases)))
		td)))))

(define qpu-low-table-aliases (lambda (tables)
	(map (coalesceNil tables '()) (lambda (td)
		(match td
			'(a _ _ _ _) a
			nil)))))

(define qpu-low-table-aliases-recursive (lambda (tables)
	(reduce (coalesceNil tables '()) (lambda (acc td)
		(match td
			'(a _ source _ _)
			(merge acc (list a)
				(if (qpp-tuple? source)
					(qpu-low-table-aliases-recursive
						(qpp-tuple-tables source))
					'()))
			acc))
		'())))

(define qpu-low-carry-table-one-scope-deeper (lambda (td local-aliases)
	(match td
		'(a s t io je)
		(list a s
			(if (qpp-tuple? t)
				(qpu-low-carry-tuple-one-scope-deeper t)
				t)
			io
			(if (nil? je) nil
				(qpu-low-add-outer-hop-for-nonlocal-refs je local-aliases)))
		td)))

(define qpu-low-carry-tuple-one-scope-deeper (lambda (tuple)
	(if (not (qpp-tuple? tuple)) tuple
		(qpp-rebuild-tuple
			(qpp-tuple-schema tuple)
			(map (coalesceNil (qpp-tuple-tables tuple) '())
				(lambda (td)
					(qpu-low-carry-table-one-scope-deeper td
						(qpu-low-table-aliases (qpp-tuple-tables tuple)))))
			(qpp-tuple-fields tuple)
			(qpp-tuple-condition tuple)
			(qpp-tuple-group tuple)
			(qpp-tuple-having tuple)
			(qpp-tuple-order tuple)
			(qpp-tuple-limit tuple)
			(qpp-tuple-offset tuple)))))

(define qpu-low-dot-ref-outside? (lambda (expr local-aliases)
	(if (list? expr)
		false
		(match (split (string expr) ".")
			(list tv _) (not (has? local-aliases tv))
			false))))

(define qpu-low-add-outer-hop-for-dot-refs (lambda (expr local-aliases)
	(match expr
		'((symbol outer) inner) (if (qpu-low-dot-ref-outside? inner local-aliases)
			(list (quote outer) expr)
			(list (quote outer) (qpu-low-add-outer-hop-for-dot-refs inner local-aliases)))
		'((quote outer) inner) (if (qpu-low-dot-ref-outside? inner local-aliases)
			(list (quote outer) expr)
			(list (quote outer) (qpu-low-add-outer-hop-for-dot-refs inner local-aliases)))
		(cons head args)
		(cons head (map (coalesceNil args '())
			(lambda (a) (qpu-low-add-outer-hop-for-dot-refs a local-aliases))))
		expr)))

(define qpu-low-carry-pipeline-rest-one-scope-deeper (lambda (tables local-aliases)
	(map (coalesceNil tables '()) (lambda (td) (match td
		'(a s t io je)
		(list a s t io
			(if (nil? je) nil
				(qpu-low-add-outer-hop-for-dot-refs je local-aliases)))
		td)))))

(define qpu-low-head-is? (lambda (head sym)
	(or (equal? head sym)
		(equal? head (symbol sym))
		(equal? head (list (quote quote) sym)))))

(define qpu-low-expr-has-call? (lambda (expr call-name)
	(match expr
		(cons head args)
		(or
			(qpu-low-head-is? head call-name)
			(qpu-low-expr-has-call? head call-name)
			(reduce (coalesceNil args '()) (lambda (acc a)
				(or acc (qpu-low-expr-has-call? a call-name))) false))
		false)))

(define qpu-low-expr-has-explicit-outer? (lambda (expr)
	(match expr
		(cons head args)
		(or
			(qpu-low-head-is? head (quote outer))
			(qpu-low-expr-has-explicit-outer? head)
			(reduce (coalesceNil args '()) (lambda (acc a)
				(or acc (qpu-low-expr-has-explicit-outer? a))) false))
		false)))

(define qpu-low-tuple-has-call? (lambda (tuple call-name)
	(or
		(qpu-low-expr-has-call? (qpp-tuple-fields tuple) call-name)
		(qpu-low-expr-has-call? (qpp-tuple-condition tuple) call-name)
		(qpu-low-expr-has-call? (qpp-tuple-group tuple) call-name)
		(qpu-low-expr-has-call? (qpp-tuple-having tuple) call-name)
		(qpu-low-expr-has-call? (qpp-tuple-order tuple) call-name)
		(qpu-low-expr-has-call? (qpp-tuple-tables tuple) call-name))))

(define qpu-low-rest-entry-depends-on-rest? (lambda (td rest-aliases)
	(match td
		'(a _ _ _ je)
		(and (not (nil? je))
			(reduce (extract_tblvars je) (lambda (found tv)
				(or found (and (not (equal? tv a)) (has? rest-aliases tv))))
				false))
		false)))

(define qpu-low-expr-refs-any-alias? (lambda (expr aliases)
	(reduce (extract_tblvars expr) (lambda (found tv)
		(or found (has? aliases tv)))
		false)))

(define qpu-low-needs-skip-outer-pipeline? (lambda (right-tuple)
	(begin
		(define tbls (coalesceNil (qpp-tuple-tables right-tuple) '()))
		(define rest-tbls (cdr tbls))
			(define rest-aliases
				(map rest-tbls (lambda (td) (match td
					'(a _ _ _ _) a
					nil))))
		(or
			(and
				(qpu-low-expr-has-explicit-outer? (qpp-tuple-condition right-tuple))
				(qpu-low-expr-refs-any-alias? (qpp-tuple-condition right-tuple)
					rest-aliases))
			(reduce rest-tbls (lambda (found td) (match td
				'(_ _ _ _ je)
				(or found
					(and
						(not (nil? je))
						(qpu-low-expr-has-explicit-outer? je)
						(qpu-low-rest-entry-depends-on-rest? td rest-aliases)))
				found))
				false)))))

(define qpu-low-push-single-alias-condition-to-tables (lambda (tuple)
	(begin
		(define cond (qpp-tuple-condition tuple))
		(if (or (nil? cond) (equal? cond true) (equal? cond (quote true)))
			tuple
			(begin
				(define table-aliases (map (coalesceNil (qpp-tuple-tables tuple) '()) (lambda (td)
					(if (or (nil? td) (< (count td) 1)) nil (nth td 0)))))
				(define nullable-table-alias? (lambda (alias_)
					(reduce (coalesceNil (qpp-tuple-tables tuple) '()) (lambda (found td)
						(or found (match td
							'(a _ _ io _) (and (equal? a alias_) io)
							false)))
						false)))
				(define push-part (lambda (state part) (begin
					(define refs (qpir-expr-column-refs part))
					(define aliases (reduce refs (lambda (acc ref) (match ref
						'(tv _) (if (or (nil? tv) (has? acc tv)) acc (merge acc (list tv)))
						acc)) '()))
					(if (and (equal? (count aliases) 1)
						(has? table-aliases (car aliases))
						(not (nullable-table-alias? (car aliases))))
						(begin
							(define target (car aliases))
							(define new-tables (map (nth state 0) (lambda (td) (match td
								'(a s t io je)
								(if (equal? a target)
									(list a s t io (qpu-low-and-cond je part))
									td)
								td))))
							(list new-tables (nth state 1)))
						(list (nth state 0) (merge (nth state 1) (list part)))))))
				(define pushed (reduce (qpu-and-conjuncts cond) push-part
					(list (qpp-tuple-tables tuple) '())))
				(qpp-rebuild-tuple
					(qpp-tuple-schema tuple)
					(nth pushed 0)
					(qpp-tuple-fields tuple)
					(qpu-and-from-conjuncts (nth pushed 1))
					(qpp-tuple-group tuple)
					(qpp-tuple-having tuple)
					(qpp-tuple-order tuple)
					(qpp-tuple-limit tuple)
					(qpp-tuple-offset tuple)))))))

/* ==================== Lowering ==================== */

/* qpu-lower-to-tuple — top-level operator dispatch. */
(define qpu-lower-to-tuple (lambda (node)
	(match (qpir-kind node)
		(quote qpir-leaf)    (qpir-leaf-7tuple node)
		(quote qpir-select)  (qpu-low-select node)
		(quote qpir-map)     (qpu-low-map node)
		(quote qpir-groupby) (qpu-low-groupby node)
		(quote qpir-topk)    (qpu-low-topk node)
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

(define qpu-low-topk (lambda (node) (begin
	(define child-tuple (qpu-lower-to-tuple (qpir-topk-child node)))
	(qpp-rebuild-tuple
		(qpp-tuple-schema child-tuple)
		(qpp-tuple-tables child-tuple)
		(qpp-tuple-fields child-tuple)
		(qpp-tuple-condition child-tuple)
		(qpp-tuple-group child-tuple)
		(qpp-tuple-having child-tuple)
		(qpir-topk-order node)
		(qpir-topk-limit node)
		(qpir-topk-offset node)))))

(define qpu-low-select (lambda (node) (begin
	(define child-tuple (qpu-lower-to-tuple (qpir-select-child node)))
	/* Apply any scalar-helper field rewrites from inline-flat scalars below us.
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
				'(a _ src io _) (if (and io
					(qpu-low-scalar-helper-alias? a)
					(not (scalar-helper-source-has-aggregate? src))) a nil)
				nil)))
			(lambda (a) (not (nil? a)))))
	(define scalar-helper-all-aliases
		(filter (map (coalesceNil (qpp-tuple-tables child-tuple) '()) (lambda (td)
			(match td
				'(a _ _ io _) (if (and io
					(qpu-low-scalar-helper-alias? a)) a nil)
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
	(define scalar-helper-alias-present? (lambda (target)
		(reduce scalar-helper-all-aliases (lambda (found alias_)
			(or found (equal?? alias_ target)))
			false)))
	(define table-alias-nullable? (lambda (target)
		(reduce (coalesceNil (qpp-tuple-tables child-tuple) '()) (lambda (found td)
			(or found (match td
				'(a _ _ io _) (and (equal?? a target) io)
				false)))
			false)))
	(define coalesced-count-helper-target (lambda (expr) (match expr
		'(op lhs rhs)
		(if (and (qpu-low-head-is? op (quote >)) (equal? rhs 0))
			(match lhs
				'(cop inner default)
				(if (and
					(or
						(qpu-low-head-is? cop (quote coalesce))
						(qpu-low-head-is? cop (quote coalesceNil)))
					(equal? default 0))
					(begin
						(define target (qpu-low-scalar-helper-value-ref-alias inner))
						(if (scalar-helper-alias-present? target) target nil))
					nil)
				nil)
			nil)
		nil)))
	(define count-helper-nullify-target (lambda (expr) (match expr
		'(head inner)
		(if (qpu-low-head-is? head (quote not))
			(coalesced-count-helper-target inner)
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
			(if (equal?? a target)
				(begin
					(define join-part (if (equal?? (coalesced-count-helper-target part) target)
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
	(define scalar-helper-field-name? (lambda (name)
		(or (equal?? name "value")
			(and (string? name)
				(>= (strlen name) 5)
				(equal? (substr name 0 5) "__kt_")))))
	(define scalar-helper-field-shape
		(reduce child-fields-pairs (lambda (acc pair) (match pair
			'(name _) (and acc (scalar-helper-field-name? name))
			false)) true))
	(define scalar-value-tuple
		(and
			(qpu-low-fields-has-name child-fields-pairs "value")
			scalar-helper-field-shape
			(equal? (count (coalesceNil (qpp-tuple-group child-tuple) '())) 0)
			(nil? (qpp-tuple-having child-tuple))))
	(define scalar-domain-tuple
		(and scalar-value-tuple
			(equal? (count (coalesceNil (qpp-tuple-tables child-tuple) '())) 1)))
	(define nullify-count-target (coalesced-count-helper-target new-pred))
	(if (or scalar-domain-tuple
		(and scalar-value-tuple (not (nil? nullify-count-target))))
		(strip-if-aggregate-select
			(qpp-rebuild-tuple
				(qpp-tuple-schema child-tuple)
				(if (nil? nullify-count-target)
					(qpp-tuple-tables child-tuple)
					(attach-predicate-to-table
						(qpp-tuple-tables child-tuple)
						nullify-count-target
						new-pred))
				(map child-fields-pairs (lambda (pair) (match pair
					'(name expr)
					(if (equal?? name "value")
						(list name (list (quote if) new-pred expr nil))
						pair)
					pair)))
				(if (nil? nullify-count-target)
					(qpp-tuple-condition child-tuple)
					(qpu-low-and-cond
						(qpp-tuple-condition child-tuple)
						new-pred))
				(qpp-tuple-group child-tuple)
				(qpp-tuple-having child-tuple)
				(qpp-tuple-order child-tuple)
				(qpp-tuple-limit child-tuple)
				(qpp-tuple-offset child-tuple))))
	(begin
		(define pred-parts (qpu-and-conjuncts new-pred))
		(define count-nullify-parts
			(filter pred-parts (lambda (part)
				(qpu-low-count-helper-value-predicate? part))))
		(define count-nullify-pred
			(qpu-and-from-conjuncts count-nullify-parts))
		(define push-source-fields
			(if (and scalar-value-tuple
				(> (count count-nullify-parts) 0))
				(map child-fields-pairs (lambda (pair) (match pair
					'(name expr)
					(if (equal?? name "value")
						(if (equal? expr count-nullify-pred)
							pair
							(list name (list (quote if)
								count-nullify-pred expr nil)))
						pair)
					pair)))
				(qpp-tuple-fields child-tuple)))
		(define pushed (reduce (qpu-and-conjuncts new-pred) (lambda (acc part)
			(begin
				(define target (scalar-push-target part))
				(if (qpu-low-count-helper-value-predicate? part)
					(if scalar-value-tuple
						(list (nth acc 0) (nth acc 1))
						(list (nth acc 0) (merge (nth acc 1) (list part))))
					(if (nil? target)
						(list (nth acc 0) (merge (nth acc 1) (list part)))
						(list (attach-predicate-to-table (nth acc 0) target part)
							(nth acc 1))))))
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
				push-source-fields
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
	/* Apply scoped scalar-helper field rewrites to map projections — helper tables may
	have been added by inline-flat below; map projections placed by lift
	may reference helper.value that needs the actual inner expr. */
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
	(define child-tables (qpp-tuple-tables child-tuple))
	(define rewritten-group-keys
		(map (coalesceNil (qpir-groupby-keys node) '())
			(lambda (key-expr)
				(qpu-low-sq-rewrites-apply-expr-scoped key-expr child-tables))))
	(define rewritten-agg-projections
		(map (coalesceNil agg-projections '()) (lambda (pair) (match pair
			'(name expr)
			(list name (qpu-low-sq-rewrites-apply-expr-scoped expr child-tables))
			pair))))
	(define rewritten-having
		(if (nil? (qpir-groupby-having node)) nil
			(qpu-low-sq-rewrites-apply-expr-scoped (qpir-groupby-having node) child-tables)))
	(define child-condition-refs-scalar-helper
		(reduce (qpir-expr-column-refs (coalesceNil (qpp-tuple-condition child-tuple) true))
			(lambda (found ref) (or found (match ref
				'(tv _) (qpu-low-scalar-helper-alias? tv)
				false)))
			false))
	(define child-scalar-helper-join-filter
		(qpu-and-from-conjuncts
			(filter
				(map (coalesceNil child-tables '()) (lambda (td) (match td
					'(alias_ _ _ is_outer_ join_expr_)
					(if (and
						is_outer_
						(qpu-low-scalar-helper-alias? alias_)
						(qpu-low-expr-has-count-helper-value-predicate?
							join_expr_))
						join_expr_
						nil)
					nil)))
				(lambda (expr) (not (nil? expr))))))
	(define child-filter-for-aggregate
		(qpu-low-and-cond
			(qpp-tuple-condition child-tuple)
			child-scalar-helper-join-filter))
	(define can-wrap-filtered-scalar-domain
		(and
			(or
				child-condition-refs-scalar-helper
				(not (nil? child-scalar-helper-join-filter)))
			(equal? (coalesceNil rewritten-group-keys '()) '())
			(nil? rewritten-having)))
	(define conditionalized-agg-projections
		(if can-wrap-filtered-scalar-domain
			(map rewritten-agg-projections (lambda (pair) (match pair
				'(name expr) (begin
					(match expr
						'(agg_head agg_inner agg_reduce agg_neutral)
						(list name
							(list agg_head
								(list (quote if)
									child-filter-for-aggregate
									agg_inner
									agg_neutral)
								agg_reduce
								agg_neutral))
						pair))
				pair)))
			rewritten-agg-projections))
	(define key-projections (nth
		(reduce rewritten-group-keys (lambda (acc key-expr)
			(begin
				(define raw-proj (qpu-low-key-projection key-expr))
				(define raw-name (nth raw-proj 0))
				(define unique-name (qpu-low-unique-projection-name raw-name
					(merge (nth acc 0) rewritten-agg-projections)))
				(list
					(merge (nth acc 0) (list (list unique-name (nth raw-proj 1))))
					true)))
			(list '() true))
		0))
	(define new-fields (merge key-projections conditionalized-agg-projections))
	(qpp-rebuild-tuple
		(qpp-tuple-schema child-tuple)
		(qpp-tuple-tables child-tuple)
		new-fields
		(if can-wrap-filtered-scalar-domain true (qpp-tuple-condition child-tuple))
		rewritten-group-keys
		rewritten-having
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
		/* Inline-flat is one physical representation for scalar dep-joins
		after Neumann has already eliminated every qpir-dep-join. When the
		right side is multi-table, aggregate, or otherwise needs its own
		scope, lower it as an aliased derived table. This is not a logical
		subquery fallback; the input is already fully unnested. */
			(begin
				(define left-aliases (map (qpp-tuple-tables left-tuple) (lambda (t)
					(if (or (nil? t) (< (count t) 1)) nil (nth t 0)))))
					(define right-has-explicit-limit-one
						(equal? (qpp-tuple-limit right-tuple) 1))
					(define right-has-exists-value-payload
						(or
							(reduce (qpp-fields-to-pairs (qpp-tuple-fields right-tuple))
								(lambda (found pair) (match pair
									'(_ expr) (or found
										(qpu-low-expr-has-exists-value-payload? expr))
									found))
								false)
							(qpu-low-expr-has-exists-value-payload?
								(qpp-tuple-condition right-tuple))
							(qpu-low-expr-has-exists-value-payload?
								(qpp-tuple-having right-tuple))
							(qpu-low-expr-has-exists-value-payload?
								(qpp-tuple-order right-tuple))))
				(if (and
					right-has-explicit-limit-one
					(not right-has-exists-value-payload)
					(qpu-low-inline-scalar-eligible? right-tuple join-pred
						rhs-alias jtype left-aliases))
					(qpu-low-join-inline-scalar left-tuple right-tuple join-pred
					rhs-alias jtype)
				(if (and
					right-has-explicit-limit-one
					(equal? jtype (quote left))
					(qpu-low-scalar-helper-alias? rhs-alias)
					(not right-has-exists-value-payload)
					(> (count (coalesceNil (qpp-tuple-tables right-tuple) '())) 1)
					(equal? (count (coalesceNil (qpp-tuple-group right-tuple) '())) 0)
					(nil? (qpp-tuple-having right-tuple))
					(not (qpu-low-tuple-has-scalar-helper-table? right-tuple))
					(not (qpu-low-tuple-has-call? right-tuple (quote scalar_scan))))
					(qpu-low-join-inline-scalar-pipeline left-tuple right-tuple join-pred
						rhs-alias jtype)
					(qpu-low-join-wrap-derived left-tuple right-tuple join-pred
						rhs-alias jtype))))))))

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

(define qpu-low-scalar-helper-alias? (lambda (alias_)
	(and
		(not (nil? alias_))
		(not (list? alias_))
		(begin
			(define alias-str (string alias_))
			(and
				(>= (strlen alias-str) 14)
				(equal? (substr alias-str 0 14) "domain_scalar_"))))))

(define qpu-low-scalar-helper-value-ref-alias (lambda (expr)
	(match expr
		'(get_column tv _ col _)
		(if (and (qpu-low-scalar-helper-alias? tv)
			(equal?? col "value")) tv nil)
		'((symbol get_column) tv _ col _)
		(if (and (qpu-low-scalar-helper-alias? tv)
			(equal?? col "value")) tv nil)
		'((quote get_column) tv _ col _)
		(if (and (qpu-low-scalar-helper-alias? tv)
			(equal?? col "value")) tv nil)
		nil)))

(define qpu-low-count-helper-positive-predicate? (lambda (expr)
	(match expr
		'(op lhs rhs)
		(if (and (qpu-low-head-is? op (quote >)) (equal? rhs 0))
			(match lhs
				'(cop inner default)
				(and (qpu-low-head-is? cop (quote coalesce))
					(equal? default 0)
					(not (nil? (qpu-low-scalar-helper-value-ref-alias inner))))
				false)
			false)
		false)))

		(define qpu-low-count-helper-value-predicate? (lambda (expr)
			(match expr
				'(head inner)
				(if (qpu-low-head-is? head (quote not))
					(qpu-low-count-helper-positive-predicate? inner)
					(qpu-low-count-helper-positive-predicate? expr))
				(qpu-low-count-helper-positive-predicate? expr))))

			(define qpu-low-expr-has-count-helper-value-predicate? (lambda (expr)
				(or
					(qpu-low-count-helper-value-predicate? expr)
					(match expr
						(cons head args)
						(or
							(qpu-low-expr-has-count-helper-value-predicate? head)
							(reduce (coalesceNil args '()) (lambda (found arg)
								(or found
									(qpu-low-expr-has-count-helper-value-predicate?
										arg)))
								false))
						false))))

			(define qpu-low-exists-value-payload? (lambda (expr)
				(match expr
					'(head cond yes no)
					(and
						(qpu-low-head-is? head (quote if))
						(equal? yes 1)
						(or (equal? no 0) (nil? no)))
					false)))

			(define qpu-low-expr-has-exists-value-payload? (lambda (expr)
				(or
					(qpu-low-exists-value-payload? expr)
					(match expr
						(cons head args)
						(or
							(qpu-low-expr-has-exists-value-payload? head)
							(reduce (coalesceNil args '()) (lambda (found arg)
								(or found
									(qpu-low-expr-has-exists-value-payload? arg)))
								false))
						false))))

		(define qpu-low-exists-helper-value-output? (lambda (expr)
			(or
				(qpu-low-count-helper-positive-predicate? expr)
				(qpu-low-exists-value-payload? expr)
				(match expr
					'(head inner reducer neutral)
					(and
						(qpu-low-head-is? head (quote aggregate))
						(qpu-low-count-helper-positive-predicate? inner))
				false))))

	(define qpu-low-expr-has-count-helper-value-predicate? (lambda (expr)
		(or
			(qpu-low-count-helper-value-predicate? expr)
			(match expr
				(cons head args)
				(or
					(qpu-low-expr-has-count-helper-value-predicate? head)
					(reduce (coalesceNil args '()) (lambda (found arg)
						(or found
							(qpu-low-expr-has-count-helper-value-predicate? arg)))
						false))
				false))))

	(define qpu-low-count-helper-predicate-alias (lambda (expr)
		(match expr
			'(head inner)
			(if (qpu-low-head-is? head (quote not))
				(qpu-low-count-helper-predicate-alias inner)
				nil)
			'(op lhs rhs)
			(if (and (qpu-low-head-is? op (quote >)) (equal? rhs 0))
				(match lhs
					'(cop inner default)
					(if (and
						(or
							(qpu-low-head-is? cop (quote coalesce))
							(qpu-low-head-is? cop (quote coalesceNil)))
						(equal? default 0))
						(qpu-low-scalar-helper-value-ref-alias inner)
						nil)
					nil)
				nil)
			nil)))

	(define qpu-low-anti-count-helper-alias-in-expr (lambda (expr)
		(coalesce
			(if (qpu-low-predicate-is-not? expr)
				(qpu-low-count-helper-predicate-alias expr)
				nil)
			(match expr
				(cons head args)
				(coalesce
					(qpu-low-anti-count-helper-alias-in-expr head)
					(reduce (coalesceNil args '()) (lambda (found arg)
						(coalesce found
							(qpu-low-anti-count-helper-alias-in-expr arg)))
						nil))
				nil))))

	(define qpu-low-anti-count-helper-alias-in-fields (lambda (fields)
		(reduce (qpp-fields-to-pairs (coalesceNil fields '())) (lambda (found pair)
			(if (not (nil? found))
				found
				(match pair
					'(_ expr)
					(qpu-low-anti-count-helper-alias-in-expr expr)
					nil)))
			nil)))

(define qpu-low-expr-has-scalar-helper-ref? (lambda (expr) (match expr
	'(get_column tv _ _ _) (qpu-low-scalar-helper-alias? tv)
	'((symbol get_column) tv _ _ _) (qpu-low-scalar-helper-alias? tv)
	'((quote get_column)  tv _ _ _) (qpu-low-scalar-helper-alias? tv)
	(cons head args) (or
		(qpu-low-expr-has-scalar-helper-ref? head)
		(reduce (coalesceNil args '()) (lambda (acc a)
			(or acc (qpu-low-expr-has-scalar-helper-ref? a))) false))
	false)))

(define qpu-low-tuple-has-scalar-helper-table? (lambda (tuple)
	(reduce (coalesceNil (qpp-tuple-tables tuple) '()) (lambda (found td)
		(or found (match td
			'(alias_ _ sub_ _ join_)
			(or
				(qpu-low-scalar-helper-alias? alias_)
				(qpu-low-expr-has-scalar-helper-ref? join_)
				(if (qpp-tuple? sub_)
					(qpu-low-tuple-has-scalar-helper-table? sub_)
					false))
			false))) false)))

(define qpu-low-field-expr-by-name (lambda (fields name)
	(reduce (qpp-fields-to-pairs (coalesceNil fields '())) (lambda (found pair)
		(if (not (nil? found)) found
			(match pair
				'(fname fexpr) (if (equal? fname name) fexpr nil)
				nil)))
		nil)))

(define qpu-low-rhs-ref-to-nested-scalar-key? (lambda (expr rhs-alias fields)
	(match expr
		'((symbol get_column) tv _ col _)
		(and (equal? tv rhs-alias)
			(begin
				(define fexpr (qpu-low-field-expr-by-name fields col))
				(and (not (nil? fexpr))
					(qpu-low-expr-has-scalar-helper-ref? fexpr))))
		'((quote get_column) tv _ col _)
		(and (equal? tv rhs-alias)
			(begin
				(define fexpr (qpu-low-field-expr-by-name fields col))
				(and (not (nil? fexpr))
					(qpu-low-expr-has-scalar-helper-ref? fexpr))))
		false)))

(define qpu-low-expr-uses-nested-scalar-key? (lambda (expr rhs-alias fields)
	(or
		(qpu-low-rhs-ref-to-nested-scalar-key? expr rhs-alias fields)
		(match expr
			(cons head args)
			(or
				(qpu-low-expr-uses-nested-scalar-key?
					head rhs-alias fields)
				(reduce (coalesceNil args '()) (lambda (found a)
					(or found
						(qpu-low-expr-uses-nested-scalar-key?
							a rhs-alias fields)))
					false))
			false))))

(define qpu-low-conjunct-uses-nested-scalar-key? (lambda (conj rhs-alias fields)
	(qpu-low-expr-uses-nested-scalar-key? conj rhs-alias fields)))

(define qpu-low-pred-uses-nested-scalar-key? (lambda (pred rhs-alias fields)
	(reduce (qpu-and-conjuncts pred) (lambda (found conj)
		(or found (qpu-low-conjunct-uses-nested-scalar-key?
			conj rhs-alias fields)))
		false)))

(define qpu-low-expand-rhs-field-refs (lambda (expr rhs-alias fields)
	(match expr
		'((symbol get_column) tv _ col _)
		(if (equal? tv rhs-alias)
			(coalesce (qpu-low-field-expr-by-name fields col) expr)
			expr)
		'((quote get_column) tv _ col _)
		(if (equal? tv rhs-alias)
			(coalesce (qpu-low-field-expr-by-name fields col) expr)
			expr)
		(cons head args)
		(cons head (map (coalesceNil args '())
			(lambda (a) (qpu-low-expand-rhs-field-refs
				a rhs-alias fields))))
		expr)))

(define qpu-low-nil-checks-rhs-col? (lambda (expr rhs-alias col)
	(match expr
		'((symbol nil?) inner)
		(begin
			(define info (qpu-low-col-ref-info inner))
			(and (not (nil? info))
				(equal? (nth info 0) rhs-alias)
				(equal? (nth info 1) col)))
		'((quote nil?) inner)
		(begin
			(define info (qpu-low-col-ref-info inner))
			(and (not (nil? info))
				(equal? (nth info 0) rhs-alias)
				(equal? (nth info 1) col)))
		false)))

(define qpu-low-domain-key-map-from-equality (lambda (expr rhs-alias)
	(match expr
		'(op lhs rhs)
		(if (qpu-low-equality-op? op)
			(begin
				(define li (qpu-low-col-ref-info lhs))
				(define ri (qpu-low-col-ref-info rhs))
				(if (and (not (nil? li)) (equal? (nth li 0) rhs-alias))
					(list (list (nth li 1) rhs))
					(if (and (not (nil? ri)) (equal? (nth ri 0) rhs-alias))
						(list (list (nth ri 1) lhs))
						'())))
			'())
		'())))

(define qpu-low-domain-key-map-from-conjunct (lambda (expr rhs-alias)
	(begin
		(define direct (qpu-low-domain-key-map-from-equality expr rhs-alias))
		(if (> (count direct) 0)
			direct
			(match expr
				(cons head args)
				(if (and (qpu-low-head-is? head (quote or))
					(equal? (count args) 2))
					(begin
						(define a (nth args 0))
						(define b (nth args 1))
						(define amap (qpu-low-domain-key-map-from-equality a rhs-alias))
						(define bmap (qpu-low-domain-key-map-from-equality b rhs-alias))
						(if (> (count amap) 0)
							(match (nth amap 0)
								'(col _)
								(if (qpu-low-nil-checks-rhs-col? b rhs-alias col)
									amap
									'())
								'())
							(if (> (count bmap) 0)
								(match (nth bmap 0)
									'(col _)
									(if (qpu-low-nil-checks-rhs-col? a rhs-alias col)
										bmap
										'())
									'())
								'())))
					'())
				'())))))

(define qpu-low-domain-key-map-from-pred (lambda (expr rhs-alias)
	(reduce (qpu-and-conjuncts expr) (lambda (acc conj)
		(merge acc
			(qpu-low-domain-key-map-from-conjunct
				conj rhs-alias)))
		'())))

(define qpu-low-rhs-self-key-ref? (lambda (expr rhs-alias key-name)
	(begin
		(define info (qpu-low-col-ref-info expr))
		(and (not (nil? info))
			(equal? (nth info 0) rhs-alias)
			(equal? (nth info 1) key-name)))))

(define qpu-low-rewrite-fields-by-key-map (lambda (fields key-map rhs-alias)
	(map (coalesceNil fields '()) (lambda (pair) (match pair
		'(name expr)
		(begin
			(define domain-key-expr
				(qpu-low-key-map-lookup key-map name))
			(list name
				(if (or (nil? domain-key-expr)
					(equal? name "__kt_id")
					(qpu-low-rhs-self-key-ref?
						domain-key-expr rhs-alias name))
					(qpu-low-rewrite-left-join-key-expr
						expr key-map rhs-alias)
					domain-key-expr)))
		pair)))))

(define qpu-low-rewrite-domain-key-expr (lambda (expr fields key-map)
	(begin
		(define replacement
			(reduce (coalesceNil fields '()) (lambda (found pair)
				(if (not (nil? found)) found
					(match pair
						'(name fexpr)
						(if (equal? expr fexpr)
							(qpu-low-key-map-lookup key-map name)
							nil)
						nil)))
				nil))
		(coalesce replacement expr))))

(define qpu-low-rewrite-domain-key-list (lambda (items fields key-map)
	(map (coalesceNil items '()) (lambda (expr)
		(qpu-low-rewrite-domain-key-expr expr fields key-map)))))

(define qpu-low-domain-key-map-aliases (lambda (key-map)
	(reduce (coalesceNil key-map '()) (lambda (acc pair) (match pair
		'(key expr)
		(if (equal? key "__kt_id")
			acc
			(merge acc (qpu-low-expr-ref-aliases expr)))
		acc)) '())))

(define qpu-low-expr-ref-aliases (lambda (expr)
	(match expr
		'((symbol outer) inner)
		(begin
			(define info
				(coalesce
					(qpu-low-dot-ref-info inner)
					(qpu-low-col-ref-info inner)))
			(if (nil? info) '()
				(list (nth info 0))))
		'((quote outer) inner)
		(begin
			(define info
				(coalesce
					(qpu-low-dot-ref-info inner)
					(qpu-low-col-ref-info inner)))
			(if (nil? info) '()
				(list (nth info 0))))
		'(head inner)
		(if (qpu-low-head-is? head (quote outer))
			(begin
				(define info
					(coalesce
						(qpu-low-dot-ref-info inner)
						(qpu-low-col-ref-info inner)))
				(if (nil? info) '()
					(list (nth info 0))))
			(map (qpir-expr-column-refs expr) (lambda (ref) (match ref
				'(tv _) tv
				nil))))
		(cons head args)
		(merge
			(qpu-low-expr-ref-aliases head)
			(reduce (coalesceNil args '()) (lambda (acc arg)
				(merge acc (qpu-low-expr-ref-aliases arg))) '()))
		(begin
			(define info (qpu-low-dot-ref-info expr))
			(if (nil? info)
				(map (qpir-expr-column-refs expr) (lambda (ref) (match ref
					'(tv _) tv
					nil)))
				(list (nth info 0)))))))

(define qpu-low-table-join-ref-aliases (lambda (tables)
	(reduce (coalesceNil tables '()) (lambda (acc td) (match td
		'(_ _ source_ _ join_expr_)
		(merge
			(merge acc (qpu-low-expr-ref-aliases join_expr_))
			(if (qpp-tuple? source_)
				(qpu-low-table-join-ref-aliases
					(qpp-tuple-tables source_))
				'()))
		acc)) '())))

(define qpu-low-tuple-join-refs-aliases? (lambda (tuple aliases)
	(reduce (qpu-low-table-join-ref-aliases
		(qpp-tuple-tables tuple)) (lambda (found alias_)
			(or found (has? aliases alias_))) false)))

(define qpu-low-filter-tables-by-aliases (lambda (tables aliases target)
	(filter (coalesceNil tables '()) (lambda (td) (match td
		'(alias_ _ _ _ _)
		(and (not (equal? alias_ target))
			(has? aliases alias_))
		false)))))

(define qpu-low-cond-refs-only-aliases? (lambda (expr aliases)
	(reduce (qpu-low-expr-ref-aliases expr) (lambda (ok alias_)
		(and ok (has? aliases alias_)))
		true)))

(define qpu-low-filter-cond-by-aliases (lambda (cond aliases)
	(qpu-and-from-conjuncts
		(filter (qpu-and-conjuncts cond) (lambda (conj)
			(qpu-low-cond-refs-only-aliases? conj aliases))))))

(define qpu-low-strip-nil-tolerant-key-cond (lambda (expr)
	(match expr
		(cons head args)
		(if (and (qpu-low-head-is? head (quote or))
			(equal? (count args) 2))
			(begin
				(define a (nth args 0))
				(define b (nth args 1))
				(match a
					'(nil-head nil-arg)
					(if (qpu-low-head-is? nil-head (quote nil?))
						b
						(match b
							'(nil-head2 nil-arg2)
							(if (qpu-low-head-is? nil-head2 (quote nil?))
								a
								(cons head (map args qpu-low-strip-nil-tolerant-key-cond)))
							(cons head (map args qpu-low-strip-nil-tolerant-key-cond))))
					(match b
						'(nil-head2 nil-arg2)
						(if (qpu-low-head-is? nil-head2 (quote nil?))
							a
							(cons head (map args qpu-low-strip-nil-tolerant-key-cond)))
						(cons head (map args qpu-low-strip-nil-tolerant-key-cond)))))
			(cons head (map (coalesceNil args '())
				qpu-low-strip-nil-tolerant-key-cond)))
		expr)))

(define qpu-low-domain-nonlocal-key-cond (lambda (key-map target)
	(qpu-and-from-conjuncts
		(filter (map (coalesceNil key-map '()) (lambda (pair) (match pair
			'(key expr)
			(if (equal? key "__kt_id")
				true
				(begin
					(define target-ref
						(list (quote get_column) target false key false))
					(list (quote equal??) expr target-ref)))
			true)))
			(lambda (conj) (not (equal? conj true)))))))

(define qpu-low-domain-key-field? (lambda (pair) (match pair
	'(name _)
	(and
		(not (nil? name))
		(not (list? name))
		(begin
			(define name-str (string name))
			(and
				(>= (strlen name-str) 5)
				(equal? (substr name-str 0 5) "__kt_"))))
	false)))

(define qpu-low-domain-key-presence-pred? (lambda (pred fields)
	(begin
		(define key-exprs
			(map (filter fields qpu-low-domain-key-field?)
				(lambda (pair) (match pair
					'(_ expr) expr
					nil))))
		(and
			(> (count key-exprs) 0)
			(reduce (qpu-and-conjuncts pred) (lambda (ok conj)
				(and ok (has? key-exprs conj)))
				true)))))

(define qpu-low-domain-collapse-limit-one? (lambda (tuple fields)
	(and
		(equal? (qpp-tuple-limit tuple) 1)
		(nil? (qpp-tuple-offset tuple))
		(equal? (coalesceNil (qpp-tuple-order tuple) '()) '())
		(equal? (coalesceNil (qpp-tuple-group tuple) '()) '())
		(nil? (qpp-tuple-having tuple))
		(> (count (filter fields qpu-low-domain-key-field?)) 0))))

(define qpu-low-limit-one-value-reducer
	(list (quote lambda)
		(list (quote acc) (quote val))
		(list (quote if)
			(list (quote nil?) (quote val))
			(quote acc)
			(quote val))))

(define qpu-low-collapse-domain-fields (lambda (fields)
	(map fields (lambda (pair) (match pair
		'(name expr)
		(if (qpu-low-domain-key-field? pair)
			pair
			(list name (list (quote aggregate) expr
				qpu-low-limit-one-value-reducer nil)))
		pair)))))

(define qpu-low-wrap-domain-value-fields (lambda (fields pred)
	(if (or (nil? pred) (equal? pred true) (equal? pred (quote true))
		(qpu-low-domain-key-presence-pred? pred fields))
		fields
		(map fields (lambda (pair) (match pair
			'(name expr)
			(if (qpu-low-domain-key-field? pair)
				pair
				(if (or (equal? expr pred)
					(qpu-low-count-helper-value-predicate? expr))
					pair
					(list name (list (quote if) pred expr nil))))
			pair))))))

(define qpu-low-predicate-is-not? (lambda (pred)
	(match pred
		'(head _) (qpu-low-head-is? head (quote not))
		false)))

(define qpu-low-strip-tautological-equalities (lambda (expr)
	(match expr
		'(head lhs rhs)
		(if (and (qpu-low-equality-op? head)
			(equal? lhs rhs))
			true
			(list head
				(qpu-low-strip-tautological-equalities lhs)
				(qpu-low-strip-tautological-equalities rhs)))
		(cons head args)
		(if (qpu-low-head-is? head (quote and))
			(qpu-and-from-conjuncts
				(filter
					(map (coalesceNil args '())
						qpu-low-strip-tautological-equalities)
					(lambda (part)
						(not (or (equal? part true)
							(equal? part (quote true)))))))
			(cons head
				(map (coalesceNil args '())
					qpu-low-strip-tautological-equalities)))
		expr)))

(define qpu-low-domain-key-group (lambda (fields)
	(reduce (filter fields qpu-low-domain-key-field?) (lambda (acc pair) (match pair
		'(_ expr)
		(if (has? acc expr) acc (merge acc (list expr)))
		acc))
		'())))

(define qpu-low-domain-key-not-nil-cond (lambda (fields)
	(qpu-and-from-conjuncts
		(map (filter fields qpu-low-domain-key-field?) (lambda (pair) (match pair
			'(_ expr)
			(list (quote not) (list (quote nil?) expr))
			true))))))

(define qpu-low-table-entry-is-outer? (lambda (tables target)
	(reduce (coalesceNil tables '()) (lambda (found td)
		(if (not (nil? found)) found
			(match td
				'(a _ t io _)
				(if (equal? a target)
					io
					(if (qpp-tuple? t)
						(qpu-low-table-entry-is-outer?
							(qpp-tuple-tables t) target)
						nil))
				nil)))
		nil)))

(define qpu-low-tuple-has-tagged-limit-one-source? (lambda (tuple)
	(reduce (coalesceNil (qpp-tuple-tables tuple) '()) (lambda (found td)
		(or found
			(match td
				'(_ _ t _ _)
				(or
					(qpu-low-scan-tagged-table? t)
					(and (qpp-tuple? t)
						(qpu-low-tuple-has-tagged-limit-one-source? t)))
				false)))
		false)))

(define qpu-low-target-source-ordered-limit-one? (lambda (tables target)
	(reduce (coalesceNil tables '()) (lambda (found td)
		(if found found
			(match td
				'(a _ t _ _)
				(if (and (equal? a target) (qpp-tuple? t))
					(or
						(qpu-low-tuple-has-tagged-limit-one-source? t)
						(and
							(equal? (qpp-tuple-limit t) 1)
							(nil? (qpp-tuple-offset t))))
					false)
				false)))
		false)))

(define qpu-low-target-source-unordered-limit-one? (lambda (tables target)
	(reduce (coalesceNil tables '()) (lambda (found td)
		(if found found
			(match td
				'(a _ t _ _)
				(if (and (equal? a target) (qpp-tuple? t))
					(and
						(equal? (qpp-tuple-limit t) 1)
						(nil? (qpp-tuple-offset t))
						(equal? (coalesceNil (qpp-tuple-order t) '()) '()))
					false)
				false)))
		false)))

(define qpu-low-list-append (lambda (left right)
	(if (equal? (coalesceNil left '()) '())
		right
		(cons (car left)
			(qpu-low-list-append (cdr left) right)))))

(define qpu-low-prioritize-table-alias (lambda (tables target)
	(if (nil? target)
		tables
		(qpu-low-list-append
			(filter (coalesceNil tables '()) (lambda (td) (match td
				'(a _ _ _ _) (equal? (string a) (string target))
				false)))
			(filter (coalesceNil tables '()) (lambda (td) (match td
				'(a _ _ _ _) (not (equal? (string a) (string target)))
				true)))))))

(define qpu-low-prioritize-scalar-helper-tables (lambda (tables)
	(qpu-low-list-append
		(filter (coalesceNil tables '()) (lambda (td) (match td
			'(a _ _ _ _) (qpu-low-scalar-helper-alias? a)
			false)))
		(filter (coalesceNil tables '()) (lambda (td) (match td
			'(a _ _ _ _) (not (qpu-low-scalar-helper-alias? a))
			true))))))

(define qpu-low-order-tables-by-join-deps (lambda (tables)
	(begin
		(define all-aliases (qpu-low-table-aliases tables))
		(define table-deps (lambda (td) (match td
			'(a _ _ _ je)
			(filter (merge
				(qpu-low-expr-ref-aliases je)
				(extract_tblvars je)) (lambda (ref)
					(and
						(not (equal? ref a))
						(has? all-aliases ref))))
			'())))
		(define table-ready? (lambda (td available)
			(reduce (table-deps td) (lambda (ok dep)
				(and ok (has? available dep)))
				true)))
		(define order-step (lambda (pending available)
			(if (equal? (coalesceNil pending '()) '())
				'()
				(begin
					(define ready (filter pending (lambda (td)
						(table-ready? td available))))
					(if (equal? ready '())
						pending
						(qpu-low-list-append ready
							(order-step
								(filter pending (lambda (td)
									(not (has? ready td))))
								(qpu-low-list-append available
									(qpu-low-table-aliases ready)))))))))
		(order-step (coalesceNil tables '()) '()))))

(define qpu-low-move-target-join-cond-to-dependency (lambda (tables target)
	(begin
		(define all-aliases (qpu-low-table-aliases tables))
		(define target-join
			(reduce (coalesceNil tables '()) (lambda (found td)
				(if (not (nil? found)) found
					(match td
						'(a _ _ _ je)
						(if (equal? a target) je nil)
						nil)))
				nil))
		(define dep-alias
			(reduce (merge
				(qpu-low-expr-ref-aliases target-join)
				(extract_tblvars target-join)) (lambda (found ref)
					(if (not (nil? found)) found
						(if (and
							(not (equal? ref target))
							(has? all-aliases ref))
							ref
							nil)))
				nil))
		(if (or (nil? target-join) (nil? dep-alias)
			(and
				(or
					(qpu-low-target-source-unordered-limit-one?
						tables target)
					(qpu-low-target-source-ordered-limit-one?
						tables target))
				(has? (merge
					(qpu-low-expr-ref-aliases target-join)
					(extract_tblvars target-join)) target))
			(qpu-low-expr-has-explicit-outer? target-join))
			tables
			(qpu-low-add-join-cond-to-table
				(map (coalesceNil tables '()) (lambda (td) (match td
					'(a s t io je)
					(if (equal? a target)
						(list a s t io nil)
						td)
					td)))
				dep-alias target-join)))))

(define qpu-low-move-scalar-helper-join-conds (lambda (tables)
	(reduce (qpu-low-table-aliases tables) (lambda (current alias_)
		(if (qpu-low-scalar-helper-alias? alias_)
			(qpu-low-move-target-join-cond-to-dependency
				current alias_)
			current))
		tables)))

(define qpu-low-nonouter-table-join-cond (lambda (tables)
	(reduce (coalesceNil tables '()) (lambda (acc td) (match td
		'(_ _ _ io je)
		(if io acc
			(qpu-low-and-cond acc je))
		acc))
		nil)))

(define qpu-low-order-ref-aliases (lambda (order)
	(reduce (coalesceNil order '()) (lambda (acc item) (match item
		'(expr _)
		(reduce (qpir-expr-column-refs expr) (lambda (aliases ref) (match ref
			'(tv _)
			(if (has? aliases tv) aliases (merge aliases (list tv)))
			aliases))
			acc)
		acc))
		'())))

(define qpu-low-fields-ref-aliases (lambda (fields)
	(reduce (coalesceNil fields '()) (lambda (acc pair) (match pair
		'(_ expr)
		(merge acc (qpu-low-expr-ref-aliases expr))
		acc))
		'())))

(define qpu-low-domain-key-repr-in-cond (lambda (key-expr target-alias cond)
	(begin
		(define direct (qpu-low-col-ref-info key-expr))
		(if (and (not (nil? direct))
			(equal? (nth direct 0) target-alias))
			key-expr
			(reduce (qpu-and-conjuncts cond) (lambda (found conj)
				(if (not (nil? found)) found
					(match conj
						'(op lhs rhs)
						(if (qpu-low-equality-op? op)
							(begin
								(define li (qpu-low-col-ref-info lhs))
								(define ri (qpu-low-col-ref-info rhs))
								(if (and (equal? lhs key-expr)
									(not (nil? ri))
									(equal? (nth ri 0) target-alias))
									rhs
									(if (and (equal? rhs key-expr)
										(not (nil? li))
										(equal? (nth li 0) target-alias))
										lhs
										nil)))
							nil)
						nil)))
				nil)))))

(define qpu-low-rewrite-domain-reprs-expr (lambda (expr repr-map)
	(begin
		(define replacement
			(reduce (coalesceNil repr-map '()) (lambda (found pair) (match pair
				'(orig repl)
				(if (and (nil? found) (equal? expr orig)) repl found)
				found))
				nil))
		(if (not (nil? replacement))
			replacement
			(match expr
				(cons head args)
				(cons
					(qpu-low-rewrite-domain-reprs-expr head repr-map)
					(map (coalesceNil args '()) (lambda (arg)
						(qpu-low-rewrite-domain-reprs-expr arg repr-map))))
				expr)))))

(define qpu-low-rewrite-domain-reprs-fields (lambda (fields repr-map)
	(map (coalesceNil fields '()) (lambda (pair) (match pair
		'(name expr)
		(list name (qpu-low-rewrite-domain-reprs-expr expr repr-map))
		pair)))))

(define qpu-low-expr-refs-alias? (lambda (expr alias_)
	(or
		(reduce (qpir-expr-column-refs expr) (lambda (found ref) (match ref
			'(tv _) (or found (equal? tv alias_))
			found))
			false)
		(match expr
			'((symbol outer) inner)
			(qpu-low-expr-refs-alias? inner alias_)
			'((quote outer) inner)
			(qpu-low-expr-refs-alias? inner alias_)
			(cons head args)
			(or
				(qpu-low-expr-refs-alias? head alias_)
				(reduce (coalesceNil args '()) (lambda (found arg)
					(or found
						(qpu-low-expr-refs-alias? arg alias_)))
					false))
			(begin
				(define info (qpu-low-dot-ref-info expr))
				(and (not (nil? info))
					(equal? (nth info 0) alias_)))))))

(define qpu-low-repr-map-excluding-local-replacements (lambda (repr-map alias_)
	(filter (coalesceNil repr-map '()) (lambda (pair) (match pair
		'(_ repl)
		(not (qpu-low-expr-refs-alias? repl alias_))
		true)))))

(define qpu-low-rewrite-domain-reprs-tables (lambda (tables repr-map)
	(map (coalesceNil tables '()) (lambda (td) (match td
		'(alias_ schema_ source_ is_outer_ join_expr_)
		(begin
			(define table-repr-map
				(qpu-low-repr-map-excluding-local-replacements
					repr-map alias_))
			(list alias_ schema_
				(if (qpp-tuple? source_)
					(qpp-rebuild-tuple
						(qpp-tuple-schema source_)
						(qpu-low-rewrite-domain-reprs-tables
							(qpp-tuple-tables source_) repr-map)
						(qpp-tuple-fields source_)
						(qpp-tuple-condition source_)
						(qpp-tuple-group source_)
						(qpp-tuple-having source_)
						(qpp-tuple-order source_)
						(qpp-tuple-limit source_)
						(qpp-tuple-offset source_))
					source_)
				is_outer_
				(qpu-low-rewrite-domain-reprs-expr
					join_expr_ table-repr-map)))
		td)))))

(define qpu-low-field-repr-map (lambda (key-fields key-reprs)
	(begin
		(define idx (newsession))
		(idx "i" 0)
		(map key-fields (lambda (pair)
			(begin
				(define pos (idx "i"))
				(idx "i" (+ pos 1))
				(define repr (nth key-reprs pos))
				(match pair
					'(_ expr) (list expr repr)
					(list nil repr))))))))

(define qpu-low-outer-dot-ref-info (lambda (expr)
	(match expr
		'((symbol outer) symname)
		(if (list? symname)
			(qpu-low-col-ref-info symname)
			(match (split (string symname) ".")
				'(tv col) (list tv col)
				nil))
		'((quote outer) symname)
		(if (list? symname)
			(qpu-low-col-ref-info symname)
			(match (split (string symname) ".")
				'(tv col) (list tv col)
				nil))
		nil)))

(define qpu-low-dot-ref-info (lambda (expr)
	(if (list? expr) nil
		(match (split (string expr) ".")
			'(tv col) (list tv col)
			nil))))

(define qpu-low-outer-dot-ref-from-info (lambda (info)
	(if (nil? info) nil
		(list (quote outer)
			(symbol (concat (nth info 0) "." (nth info 1)))))))

(define qpu-low-local-vs-outer-repr-map (lambda (pred preferred-aliases)
	(reduce (qpu-and-conjuncts pred) (lambda (acc conj)
		(match conj
			'(op lhs rhs)
			(if (qpu-low-equality-op? op)
				(begin
					(define li
						(coalesce
							(qpu-low-col-ref-info lhs)
							(qpu-low-dot-ref-info lhs)))
					(define ri
						(coalesce
							(qpu-low-col-ref-info rhs)
							(qpu-low-dot-ref-info rhs)))
					(define lo (qpu-low-outer-dot-ref-info lhs))
					(define ro (qpu-low-outer-dot-ref-info rhs))
					(if (and (not (nil? li)) (has? preferred-aliases (nth li 0))
						(or
							(not (nil? ro))
							(and (not (nil? ri))
								(not (has? preferred-aliases (nth ri 0))))))
						(merge acc (list (list rhs lhs)))
						(if (and (not (nil? ri)) (has? preferred-aliases (nth ri 0))
							(or
								(not (nil? lo))
								(and (not (nil? li))
									(not (has? preferred-aliases (nth li 0))))))
							(merge acc (list (list lhs rhs)))
							acc)))
				acc)
			acc))
		'())))

(define qpu-low-nonlocal-key-cond-repr-map (lambda (pred preferred-aliases)
	(reduce (qpu-and-conjuncts pred) (lambda (acc conj)
		(match conj
			'(op lhs rhs)
			(if (qpu-low-equality-op? op)
				(begin
					(define li (qpu-low-col-ref-info lhs))
					(define ri (qpu-low-col-ref-info rhs))
					(define lo (qpu-low-outer-dot-ref-info lhs))
					(define ro (qpu-low-outer-dot-ref-info rhs))
					(if (and (not (nil? li)) (has? preferred-aliases (nth li 0)))
						(begin
							(define rhs-info
								(coalesce ro
									(if (and (not (nil? ri))
										(not (has? preferred-aliases (nth ri 0))))
										ri
										nil)))
							(if (nil? rhs-info) acc
								(merge acc
									(list
										(list
											(qpu-low-outer-dot-ref-from-info rhs-info)
											lhs)))))
						(if (and (not (nil? ri)) (has? preferred-aliases (nth ri 0)))
							(begin
								(define lhs-info
									(coalesce lo
										(if (and (not (nil? li))
											(not (has? preferred-aliases (nth li 0))))
											li
											nil)))
								(if (nil? lhs-info) acc
									(merge acc
										(list
											(list
												(qpu-low-outer-dot-ref-from-info lhs-info)
												rhs)))))
							acc)))
				acc)
			acc))
		'())))

(define qpu-low-expr-contains-ref-info? (lambda (expr info)
	(match expr
		'((symbol get_column) tv _ col _)
		(and (equal? tv (nth info 0)) (equal? col (nth info 1)))
		'((quote get_column) tv _ col _)
		(and (equal? tv (nth info 0)) (equal? col (nth info 1)))
		'((symbol outer) inner)
		(begin
			(define inner-info
				(coalesce
					(qpu-low-dot-ref-info inner)
					(qpu-low-col-ref-info inner)))
			(and (not (nil? inner-info))
				(equal? inner-info info)))
		'((quote outer) inner)
		(begin
			(define inner-info
				(coalesce
					(qpu-low-dot-ref-info inner)
					(qpu-low-col-ref-info inner)))
			(and (not (nil? inner-info))
				(equal? inner-info info)))
		(cons head args)
		(or
			(qpu-low-expr-contains-ref-info? head info)
			(reduce (coalesceNil args '()) (lambda (found arg)
				(or found
					(qpu-low-expr-contains-ref-info? arg info)))
				false))
		(begin
			(define dot-info (qpu-low-dot-ref-info expr))
			(and (not (nil? dot-info))
				(equal? dot-info info))))))

(define qpu-low-fields-contain-ref-info? (lambda (fields info)
	(reduce (coalesceNil fields '()) (lambda (found pair) (match pair
		'(_ expr)
		(or found (qpu-low-expr-contains-ref-info? expr info))
		found))
		false)))

(define qpu-low-outer-dot-refs (lambda (expr)
	(match expr
		'((symbol outer) inner)
		(begin
			(define info
				(coalesce
					(qpu-low-dot-ref-info inner)
					(qpu-low-col-ref-info inner)))
			(if (nil? info) '()
				(list
					(qpu-low-outer-dot-ref-from-info info))))
		'((quote outer) inner)
		(begin
			(define info
				(coalesce
					(qpu-low-dot-ref-info inner)
					(qpu-low-col-ref-info inner)))
			(if (nil? info) '()
				(list
					(qpu-low-outer-dot-ref-from-info info))))
		(cons head args)
		(merge
			(qpu-low-outer-dot-refs head)
			(reduce (coalesceNil args '()) (lambda (acc arg)
				(merge acc (qpu-low-outer-dot-refs arg)))
				'()))
		'())))

(define qpu-low-local-id-field-expr (lambda (fields rhs-alias)
	(reduce (coalesceNil fields '()) (lambda (found pair)
		(if (not (nil? found)) found
			(match pair
				'(_ expr)
				(begin
					(define info
						(coalesce
							(qpu-low-col-ref-info expr)
							(qpu-low-dot-ref-info expr)))
					(if (and (not (nil? info))
						(equal? (nth info 0) rhs-alias))
						expr
						nil))
				nil)))
		nil)))

(define qpu-low-missing-outer-key-repr-map (lambda (cond carried-fields local-fields rhs-alias allowed-aliases)
	(begin
		(define local-key-expr
			(qpu-low-local-id-field-expr local-fields rhs-alias))
		(define local-key-for-outer-ref (lambda (outer-ref)
			(reduce (qpu-and-conjuncts cond) (lambda (found conj)
				(if (not (nil? found))
					found
					(match conj
						'(op lhs rhs)
						(if (qpu-low-equality-op? op)
							(if (equal? lhs outer-ref)
								(coalesce
									(if (nil? (qpu-low-col-ref-info rhs)) nil rhs)
									(if (nil? (qpu-low-dot-ref-info rhs)) nil rhs)
									nil)
								(if (equal? rhs outer-ref)
									(coalesce
										(if (nil? (qpu-low-col-ref-info lhs)) nil lhs)
										(if (nil? (qpu-low-dot-ref-info lhs)) nil lhs)
										nil)
									nil))
							nil)
						nil)))
				nil)))
		(if (nil? local-key-expr) '()
			(reduce (qpu-low-outer-dot-refs cond) (lambda (acc outer-ref)
				(begin
					(define info (qpu-low-outer-dot-ref-info outer-ref))
					(if (or (nil? info)
						(and
							(not (equal? (coalesceNil allowed-aliases '()) '()))
							(not (has? allowed-aliases (nth info 0))))
						(qpu-low-fields-contain-ref-info? carried-fields info))
						acc
						(begin
							(define local-repl
								(coalesce
									(local-key-for-outer-ref outer-ref)
									local-key-expr))
							(merge acc
								(list (list outer-ref local-repl)))))))
				'())))))

(define qpu-low-domain-key-map-ref-aliases (lambda (key-map)
	(reduce (coalesceNil key-map '()) (lambda (acc pair) (match pair
		'(_ expr)
		(merge acc (qpu-low-expr-ref-aliases expr))
		acc))
		'())))

(define qpu-low-domain-key-repr-map (lambda (fields key-map)
	(reduce (coalesceNil fields '()) (lambda (acc pair) (match pair
		'(name expr)
		(begin
			(define key-expr
				(qpu-low-key-map-lookup key-map name))
			(if (or (nil? key-expr) (equal? key-expr expr))
				acc
				(begin
					(define info
						(coalesce
							(qpu-low-col-ref-info key-expr)
							(qpu-low-dot-ref-info key-expr)
							(qpu-low-outer-dot-ref-info key-expr)))
					(define outer-expr
						(qpu-low-outer-dot-ref-from-info info))
					(merge acc
						(if (nil? outer-expr)
							(list (list key-expr expr))
							(list (list key-expr expr)
								(list outer-expr expr)))))))
		acc))
		'())))

(define qpu-low-tag-domain-ordered-limit-one (lambda (tuple fields)
	(begin
		(define lim (qpp-tuple-limit tuple))
		(define off (qpp-tuple-offset tuple))
		(define ord (coalesceNil (qpp-tuple-order tuple) '()))
		(define key-fields (filter fields qpu-low-domain-key-field?))
		(define partition-key-fields
			(filter key-fields (lambda (pair) (match pair
				'(_ expr)
				(not (qpu-low-expr-has-scalar-helper-ref? expr))
				true))))
		(define order-aliases (qpu-low-order-ref-aliases ord))
		(if (or
			(not (equal? lim 1))
			(not (nil? off))
			(equal? (count ord) 0)
			(not (equal? (count order-aliases) 1))
			(equal? (count partition-key-fields) 0)
			(> (count (coalesceNil (qpp-tuple-group tuple) '())) 0)
			(not (nil? (qpp-tuple-having tuple))))
			tuple
			(begin
				(define target-alias (nth order-aliases 0))
				(define key-reprs
					(filter
						(map key-fields (lambda (pair) (match pair
							'(_ expr)
							(qpu-low-domain-key-repr-in-cond
								expr target-alias
								(qpp-tuple-condition tuple))
							nil)))
						(lambda (expr) (not (nil? expr)))))
				(if (not (equal? (count key-reprs) (count key-fields)))
					tuple
					(begin
						(define repr-map
							(qpu-low-field-repr-map key-fields key-reprs))
						(define partition-key-reprs
							(filter
								(map partition-key-fields (lambda (pair) (match pair
									'(_ expr)
									(qpu-low-domain-key-repr-in-cond
										expr target-alias
										(qpp-tuple-condition tuple))
									nil)))
								(lambda (expr) (not (nil? expr)))))
						(define rewritten-fields
							(qpu-low-rewrite-domain-reprs-fields
								(qpp-tuple-fields tuple) repr-map))
						(define rewritten-condition
							(qpu-low-rewrite-domain-reprs-expr
								(qpp-tuple-condition tuple) repr-map))
						(define partition-order
							(map partition-key-reprs (lambda (expr)
								(list expr (quote <)))))
						(define tagged-entry
							(reduce (coalesceNil (qpp-tuple-tables tuple) '()) (lambda (found td) (match td
								'(td-alias td-schema td-tname td-isOuter td-je)
								(if (and (nil? found)
									(equal? td-alias target-alias)
									(not (qpp-tuple? td-tname))
									(not (qpu-low-scan-tagged-table? td-tname)))
									(list td-alias td-schema
										(make_scan_tagged_table
											td-tname
											(merge partition-order ord)
											1 nil
											(count partition-order)
											1)
										td-isOuter td-je)
									found)
								found))
								nil))
						(define referenced-aliases
							(merge
								(list target-alias)
								(merge
									(qpu-low-fields-ref-aliases
										rewritten-fields)
									(qpu-low-expr-ref-aliases
										rewritten-condition))))
						(define tagged-tables
							(if (nil? tagged-entry)
								(qpp-tuple-tables tuple)
								(cons tagged-entry
									(filter (coalesceNil (qpp-tuple-tables tuple) '()) (lambda (td) (match td
										'(td-alias _ _ _ _)
										(and
											(not (equal? td-alias target-alias))
											(has? referenced-aliases td-alias))
										true))))))
						(define remaining-aliases
							(qpu-low-table-aliases tagged-tables))
						(define cond-aliases-without-nil (lambda (conj)
							(filter (qpu-low-expr-ref-aliases conj)
								(lambda (a) (not (nil? a))))))
						(define cond-moves-to-alias? (lambda (conj alias_)
							(begin
								(define aliases (cond-aliases-without-nil conj))
								(and
									(has? aliases target-alias)
									(has? aliases alias_)
									(reduce aliases (lambda (ok a)
										(and ok (or (equal? a target-alias)
											(equal? a alias_))))
										true)))))
						(define cond-moves-to-any-nontarget? (lambda (conj)
							(reduce remaining-aliases (lambda (found alias_)
								(or found
									(and (not (equal? alias_ target-alias))
										(cond-moves-to-alias? conj alias_))))
								false)))
						(define tagged-tables-with-joins
							(map (coalesceNil tagged-tables '()) (lambda (td) (match td
								'(td-alias td-schema td-tname td-isOuter td-je)
								(if (equal? td-alias target-alias)
									td
									(begin
										(define moved-cond
											(qpu-and-from-conjuncts
												(filter (qpu-and-conjuncts rewritten-condition)
													(lambda (conj)
														(cond-moves-to-alias? conj td-alias)))))
										(list td-alias td-schema td-tname td-isOuter
											(qpu-low-and-cond td-je moved-cond))))
								td))))
						(define final-condition
							(qpu-low-filter-cond-by-aliases
								(qpu-and-from-conjuncts
									(filter (qpu-and-conjuncts rewritten-condition)
										(lambda (conj)
											(not (cond-moves-to-any-nontarget? conj)))))
								remaining-aliases))
						(qpp-rebuild-tuple
							(qpp-tuple-schema tuple)
							tagged-tables-with-joins
							rewritten-fields
							final-condition
							(qpp-tuple-group tuple)
							(qpp-tuple-having tuple)
							'()
							nil
							nil))))))))

(define qpu-low-domain-left-tuple-for-target (lambda (left-tuple right-tuple target pred)
	(begin
		(define left-aliases
			(qpu-low-table-aliases (qpp-tuple-tables left-tuple)))
		(define needed-aliases
			(merge_unique
				(list
					(qpu-low-domain-key-map-aliases
						(qpu-low-domain-key-map-from-pred pred target))
					(qpu-low-table-join-ref-aliases
						(qpp-tuple-tables right-tuple)))))
		(define domain-left-tables-raw
			(merge
				(qpu-low-filter-tables-by-aliases
					(qpp-tuple-tables left-tuple) needed-aliases target)
				(qpu-low-filter-tables-by-aliases
					(qpp-tuple-tables right-tuple) needed-aliases target)))
		(define domain-left-aliases
			(qpu-low-table-aliases domain-left-tables-raw))
		(define domain-left-tables
			(map domain-left-tables-raw (lambda (td) (match td
				'(a s t io je)
				(list a s t io
					(qpu-and-from-conjuncts
						(filter
							(qpu-and-conjuncts
								(qpu-low-filter-cond-by-aliases
									je domain-left-aliases))
							(lambda (conj)
								(not (qpu-low-expr-refs-alias?
									conj target))))))
				td))))
		(qpp-rebuild-tuple
			(qpp-tuple-schema right-tuple)
			domain-left-tables
			'()
			(qpu-low-filter-cond-by-aliases
				(qpp-tuple-condition left-tuple) needed-aliases)
			nil
			nil
			'()
			nil
			nil))))

		(define qpu-low-domainize-target-source (lambda (tables target left-tuple right-tuple pred)
			(map (coalesceNil tables '()) (lambda (td) (match td
				'(alias_ schema_ source_ is_outer_ join_expr_)
				(if (and (equal? alias_ target) (qpp-tuple? source_))
					(begin
							(define domain-left-base
								(qpu-low-domain-left-tuple-for-target
									left-tuple right-tuple target pred))
							(define domain-left-base-aliases
								(qpu-low-table-aliases
									(qpp-tuple-tables domain-left-base)))
							(define inherited-left-tables
								(filter (coalesceNil (qpp-tuple-tables left-tuple) '())
									(lambda (itd) (match itd
										'(ia _ _ _ _)
										(and
											(not (qpu-low-scalar-helper-alias? ia))
											(not (has? domain-left-base-aliases ia)))
										false))))
							(define inherited-domain-tables-raw
								(merge
									(qpp-tuple-tables domain-left-base)
									inherited-left-tables))
							(define inherited-domain-aliases
								(qpu-low-table-aliases
									inherited-domain-tables-raw))
							(define domain-left
								(qpp-rebuild-tuple
									(qpp-tuple-schema domain-left-base)
									(map inherited-domain-tables-raw (lambda (itd) (match itd
										'(ia is it ii ij)
										(list ia is it ii
											(qpu-low-filter-cond-by-aliases
												ij inherited-domain-aliases))
										itd)))
									'()
									(qpu-low-filter-cond-by-aliases
										(qpu-low-and-cond
											(qpp-tuple-condition domain-left-base)
											(qpp-tuple-condition left-tuple))
										inherited-domain-aliases)
									nil
									nil
									'()
									nil
									nil))
						(list alias_ schema_
							(qpu-low-domainize-nested-scalar-derived
								domain-left source_ target pred)
					is_outer_ join_expr_))
			td)
		td)))))

	(define qpu-low-domainize-nested-scalar-derived (lambda (left-tuple right-tuple rhs-alias pred)
		(begin
			(define right-fields (qpp-fields-to-pairs (qpp-tuple-fields right-tuple)))
			(define domain-pred
				(qpu-low-and-cond pred (qpp-tuple-condition right-tuple)))
			(define domain-key-map
				(qpu-low-domain-key-map-from-pred domain-pred rhs-alias))
		(define right-scalar-helper-source-exists-output? (lambda (alias_)
			(reduce (coalesceNil (qpp-tuple-tables right-tuple) '()) (lambda (found td) (match td
				'(a _ source _ _)
				(or found
					(and
						(equal? a alias_)
						(qpp-tuple? source)
						(begin
							(define source-value
								(qpu-low-field-expr-by-name
									(qpp-tuple-fields source) "value"))
							(qpu-low-exists-helper-value-output?
								source-value))))
				found))
				false)))
		(define right-expr-projects-derived-exists-helper? (lambda (expr)
			(match expr
				'((symbol get_column) alias_ _ col _)
				(and
					(equal?? col "value")
					(right-scalar-helper-source-exists-output? alias_))
				'((quote get_column) alias_ _ col _)
				(and
					(equal?? col "value")
					(right-scalar-helper-source-exists-output? alias_))
				false)))
			(define right-projects-direct-count-helper-value?
				(reduce right-fields (lambda (found pair) (match pair
					'(_ expr)
						(or found
							(qpu-low-count-helper-value-predicate? expr)
							(qpu-low-expr-has-exists-value-payload? expr))
					found))
					false))
		(define right-projects-derived-count-helper-value?
			(reduce right-fields (lambda (found pair) (match pair
				'(_ expr)
				(or found
					(right-expr-projects-derived-exists-helper? expr))
				found))
				false))
		(define right-projects-anti-count-helper-value?
			(not (nil? (qpu-low-anti-count-helper-alias-in-fields right-fields))))
			(define right-projects-count-helper-value?
				(or
					right-projects-direct-count-helper-value?
					right-projects-derived-count-helper-value?
					right-projects-anti-count-helper-value?))
			(define left-refs-rhs-alias?
				(or
					(qpu-low-expr-refs-alias?
						(qpp-tuple-fields left-tuple) rhs-alias)
					(qpu-low-expr-refs-alias?
						(qpp-tuple-condition left-tuple) rhs-alias)
					(qpu-low-expr-refs-alias?
						(qpp-tuple-having left-tuple) rhs-alias)
					(qpu-low-expr-refs-alias?
						(qpp-tuple-order left-tuple) rhs-alias)))
			(define top-down-nested-projection-domain?
				(and
					(> (count (coalesceNil domain-key-map '())) 0)
					(equal? (qpp-tuple-limit right-tuple) 1)
					(nil? (qpp-tuple-offset right-tuple))
					(equal? (coalesceNil (qpp-tuple-order right-tuple) '()) '())
					right-projects-count-helper-value?
					(not (qpu-low-expr-has-scalar-helper-ref?
					(qpp-tuple-condition right-tuple)))))
			(define top-down-grouped-domain?
				(and
					(> (count (coalesceNil domain-key-map '())) 0)
					left-refs-rhs-alias?
					(or
						(> (count (coalesceNil (qpp-tuple-group right-tuple) '())) 0)
					(not (nil? (qpp-tuple-having right-tuple)))
					(qpu-low-fields-have-aggregate?
						(qpp-tuple-fields right-tuple))
					top-down-nested-projection-domain?)))
		(define top-down-domain-key-name? (lambda (name)
			(reduce (coalesceNil domain-key-map '()) (lambda (found pair) (match pair
				'(key-name _)
				(or found (equal? key-name name))
				found))
				false)))
		(define top-down-guard-aggregate-value (lambda (expr pred)
			(match expr
				'(head inner reducer neutral)
				(if (qpu-low-head-is? head (quote aggregate))
					(list head
						(list (quote if) pred inner
							(if (nil? neutral) nil neutral))
						reducer neutral)
					(list (quote if) pred expr nil))
				(list (quote if) pred expr nil))))
		(define qpu-low-wrap-domain-value-fields-for-domain (lambda (fields pred)
			(if (not top-down-grouped-domain?)
				(qpu-low-wrap-domain-value-fields fields pred)
				(if (or (nil? pred) (equal? pred true) (equal? pred (quote true)))
					fields
					(map fields (lambda (pair) (match pair
						'(name expr)
						(if (or
							(top-down-domain-key-name? name)
							(equal? expr pred)
							(qpu-low-count-helper-value-predicate? expr))
							pair
							(list name
								(top-down-guard-aggregate-value expr pred)))
						pair)))))))
				(define domain-left-tuple
					(if top-down-grouped-domain?
						(begin
							(define top-down-left-tables
								(filter (coalesceNil (qpp-tuple-tables left-tuple) '())
									(lambda (td) (match td
										'(a _ _ _ _) (not (qpu-low-scalar-helper-alias? a))
										true))))
							(define top-down-left-aliases
								(qpu-low-table-aliases top-down-left-tables))
							(define top-down-left-tables-local
								(map top-down-left-tables (lambda (td) (match td
									'(a s t io je)
									(list a s t io
										(qpu-and-from-conjuncts
											(filter
												(qpu-and-conjuncts
													(qpu-low-filter-cond-by-aliases
														je top-down-left-aliases))
												(lambda (conj)
													(not (qpu-low-expr-refs-alias?
														conj rhs-alias))))))
									td))))
							(qpp-rebuild-tuple
								(qpp-tuple-schema right-tuple)
								top-down-left-tables-local
								'()
								(qpu-low-filter-cond-by-aliases
									(qpp-tuple-condition left-tuple)
									top-down-left-aliases)
								nil
								nil
								'()
								nil
								nil))
						(qpu-low-domain-left-tuple-for-target
							left-tuple right-tuple rhs-alias pred)))
		(define domain-fields-raw
			(qpu-low-rewrite-fields-by-key-map
				right-fields domain-key-map rhs-alias))
			(define raw-internal-pred-base
				(qpu-low-expand-rhs-field-refs domain-pred rhs-alias right-fields))
		(define local-source-ref-for-key (lambda (key-name)
			(begin
				(define source-col (qpu-low-kt-ref-source-col key-name))
				(if (equal? source-col key-name)
					nil
					(begin
						(define helper-ref
							(reduce (coalesceNil (qpp-tuple-tables right-tuple) '()) (lambda (found td)
								(if (not (nil? found))
									found
									(match td
										'(local-alias _ local-source _ _)
										(if (qpp-tuple? local-source)
											(begin
												(define local-field
													(qpu-low-field-expr-by-name
														(qpp-tuple-fields local-source)
														source-col))
												(if (nil? local-field)
													nil
													(list (quote get_column) local-alias false source-col false)))
											nil)
										nil)))
								nil))
						(if (not (nil? helper-ref))
							helper-ref
							nil))))))
		(define early-domain-key-local-repr-map
			(if top-down-grouped-domain? '()
				(reduce (coalesceNil domain-key-map '()) (lambda (acc pair) (match pair
					'(key-name key-expr)
					(begin
						(define key-ref
							(list (quote get_column) rhs-alias false key-name false))
						(define outer-key-ref
							(qpu-low-outer-dot-ref-from-info
								(list rhs-alias key-name)))
						(define key-ref-match? (lambda (expr)
							(or
								(equal? expr key-ref)
								(equal? expr outer-key-ref))))
						(define local-repl
							(coalesce
								(reduce (qpu-and-conjuncts raw-internal-pred-base) (lambda (found conj)
									(if (not (nil? found))
										found
										(match conj
											'(op lhs rhs)
											(if (qpu-low-equality-op? op)
												(if (and (key-ref-match? lhs)
													(not (nil? (qpu-low-col-ref-info rhs)))
													(has? (qpu-low-table-aliases (qpp-tuple-tables right-tuple))
														(nth (qpu-low-col-ref-info rhs) 0)))
													rhs
													(if (and (key-ref-match? rhs)
														(not (nil? (qpu-low-col-ref-info lhs)))
														(has? (qpu-low-table-aliases (qpp-tuple-tables right-tuple))
															(nth (qpu-low-col-ref-info lhs) 0)))
														lhs
														nil))
												nil)
											nil)))
									nil)
								(local-source-ref-for-key key-name)))
						(if (nil? local-repl)
							acc
							(begin
								(define key-info
									(coalesce
										(qpu-low-col-ref-info key-expr)
										(qpu-low-dot-ref-info key-expr)
										(qpu-low-outer-dot-ref-info key-expr)))
								(define outer-key-expr
									(qpu-low-outer-dot-ref-from-info key-info))
								(merge acc
									(if (nil? outer-key-expr)
										(list (list key-expr local-repl))
										(list (list key-expr local-repl)
											(list outer-key-expr local-repl)))))))
					acc))
					'())))
		(define early-local-outer-repr-map
			(merge
				early-domain-key-local-repr-map
				(if top-down-grouped-domain? '()
					(qpu-low-local-vs-outer-repr-map
						raw-internal-pred-base
						(qpu-low-table-aliases
							(qpp-tuple-tables right-tuple))))))
		(define domain-fields
			(if (equal? early-local-outer-repr-map '())
				domain-fields-raw
				(qpu-low-rewrite-domain-reprs-fields
					domain-fields-raw early-local-outer-repr-map)))
		(define domain-key-repr-map
			(qpu-low-domain-key-repr-map
				domain-fields domain-key-map))
		(define raw-internal-pred
			(if (equal? domain-key-repr-map '())
				raw-internal-pred-base
				(qpu-low-rewrite-domain-reprs-expr
					raw-internal-pred-base domain-key-repr-map)))
		(define local-outer-repr-map
			(if top-down-grouped-domain? '()
				(qpu-low-local-vs-outer-repr-map
					raw-internal-pred
					(qpu-low-table-aliases
						(qpp-tuple-tables right-tuple)))))
		(define nonlocal-key-cond-repr-map
			(if top-down-grouped-domain? '()
				(qpu-low-nonlocal-key-cond-repr-map
					raw-internal-pred
					(qpu-low-table-aliases
						(qpp-tuple-tables right-tuple)))))
		(define domain-key-local-repr-map
			(if top-down-grouped-domain? '()
				(qpu-low-domain-key-repr-map
					right-fields domain-key-map)))
		(define domain-key-ref-aliases
			(qpu-low-domain-key-map-ref-aliases
				domain-key-map))
		(define raw-internal-pred-effective
			(if (equal? local-outer-repr-map '())
				raw-internal-pred
				(qpu-low-rewrite-domain-reprs-expr
					raw-internal-pred local-outer-repr-map)))
		(define merged-tables-raw
			(merge (qpp-tuple-tables domain-left-tuple)
				(qpp-tuple-tables right-tuple)))
		(define merged-tables
			(if (equal? domain-key-local-repr-map '())
				merged-tables-raw
				(qpu-low-rewrite-domain-reprs-tables
					merged-tables-raw domain-key-local-repr-map)))
		(define domain-aliases
			(qpu-low-table-aliases merged-tables))
		(define internal-pred
			(qpu-low-filter-cond-by-aliases
				(qpu-low-strip-nil-tolerant-key-cond raw-internal-pred-effective)
				domain-aliases))
		(define internal-target
			(qpu-low-first-scalar-helper-ref-alias internal-pred))
		(define internal-target-key-map
			(if (nil? internal-target) '()
				(qpu-low-domain-key-map-from-pred
					internal-pred internal-target)))
		(define internal-target-repr-map
			(if (or (nil? internal-target)
				(not (qpu-low-target-source-unordered-limit-one?
					merged-tables internal-target))) '()
				(map internal-target-key-map (lambda (pair) (match pair
					'(key expr)
					(list expr
						(list (quote get_column) internal-target false key false))
					pair)))))
		(define domain-fields-effective
			(if (equal? internal-target-repr-map '())
				domain-fields
				(qpu-low-rewrite-domain-reprs-fields
					domain-fields internal-target-repr-map)))
		(define internal-pred-effective-base
			(if (equal? internal-target-repr-map '())
				internal-pred
				(qpu-low-rewrite-domain-reprs-expr
					internal-pred internal-target-repr-map)))
		(define internal-pred-effective
			internal-pred-effective-base)
		(define internal-target-ordered-limit-one
			(and (not (nil? internal-target))
				(qpu-low-target-source-ordered-limit-one?
					merged-tables internal-target)))
		(define internal-target-key-fields
			(if (nil? internal-target) '()
				(filter
					(map (coalesceNil internal-target-key-map '()) (lambda (pair) (match pair
						'(key-name _)
						(begin
							(define key-str (string key-name))
							(define source-col (qpu-low-kt-ref-source-col key-str))
							(define field-name (if (and
								(>= (strlen key-str) 5)
								(equal? (substr key-str 0 5) "__kt_"))
								key-str
								(concat "__kt_" source-col)))
							(if (or
								(and internal-target-ordered-limit-one
									(equal?? key-name "value"))
								(qpu-low-fields-has-name domain-fields-effective field-name))
								nil
								(list field-name
									(list (quote get_column) internal-target false key-name false))))
						nil)))
					(lambda (field) (not (nil? field))))))
		(define domain-fields-with-internal-keys
			(merge domain-fields-effective internal-target-key-fields))
		(define conditioned-domain-fields
			(if (nil? internal-target)
				domain-fields-with-internal-keys
				(qpu-low-wrap-domain-value-fields-for-domain
					domain-fields-with-internal-keys internal-pred-effective)))
		(define collapse-limit-one
			(qpu-low-domain-collapse-limit-one?
				right-tuple conditioned-domain-fields))
		(define output-fields
			(if collapse-limit-one
				(qpu-low-collapse-domain-fields conditioned-domain-fields)
				conditioned-domain-fields))
		(define output-group
			(if collapse-limit-one
				(qpu-low-domain-key-group conditioned-domain-fields)
				(qpu-low-rewrite-domain-key-list
					(qpp-tuple-group right-tuple)
					right-fields domain-key-map)))
		(define internal-target-nonlocal-cond
			(if (nil? internal-target) nil
				(qpu-low-domain-nonlocal-key-cond
					internal-target-key-map internal-target)))
		(define target-safe-repr-map (lambda (repr-map target)
			(filter (coalesceNil repr-map '()) (lambda (pair) (match pair
				'(_ repl)
				(not (qpu-low-expr-refs-alias? repl target))
				true)))))
		(define internal-target-safe-repr-map
			(if (nil? internal-target)
				'()
				(target-safe-repr-map
					internal-target-repr-map internal-target)))
		(define internal-target-nonlocal-cond-repr
			(if (equal? internal-target-safe-repr-map '())
				internal-target-nonlocal-cond
				(qpu-low-rewrite-domain-reprs-expr
					internal-target-nonlocal-cond
					internal-target-safe-repr-map)))
		(define missing-outer-key-repr-map
			(qpu-low-missing-outer-key-repr-map
				internal-target-nonlocal-cond-repr
				domain-fields
				right-fields
				rhs-alias
				domain-key-ref-aliases))
		(define missing-outer-key-fields
			(reduce (coalesceNil missing-outer-key-repr-map '()) (lambda (acc pair) (match pair
				'(_ repl)
				(begin
					(define repl-info
						(coalesce
							(qpu-low-col-ref-info repl)
							(qpu-low-dot-ref-info repl)))
					(if (nil? repl-info)
						acc
						(begin
							(define repl-col (nth repl-info 1))
							(define repl-col-str (string repl-col))
							(define source-col (qpu-low-kt-ref-source-col repl-col-str))
							(define field-name (if (and
								(>= (strlen repl-col-str) 5)
								(equal? (substr repl-col-str 0 5) "__kt_"))
								repl-col-str
								(concat "__kt_" source-col)))
							(if (qpu-low-fields-has-name
								(merge domain-fields-with-internal-keys acc)
								field-name)
								acc
								(merge acc (list (list field-name repl)))))))
				acc))
				'()))
		(define domain-fields-final-base
			(merge domain-fields-with-internal-keys
				missing-outer-key-fields))
		(define nested-domain-pred
			(qpu-low-rewrite-domain-reprs-expr
				internal-pred
				missing-outer-key-repr-map))
		(define nested-target-domain-pred
			(if internal-target-ordered-limit-one
				internal-target-nonlocal-cond-repr
				nested-domain-pred))
		(define internal-target-nonlocal-cond-effective
			(if (and
				(equal? nonlocal-key-cond-repr-map '())
				(equal? domain-key-local-repr-map '()))
				internal-target-nonlocal-cond-repr
				(qpu-low-rewrite-domain-reprs-expr
					(qpu-low-rewrite-domain-reprs-expr
						(qpu-low-rewrite-domain-reprs-expr
							internal-target-nonlocal-cond-repr
							(if (nil? internal-target)
								nonlocal-key-cond-repr-map
								(target-safe-repr-map
									nonlocal-key-cond-repr-map internal-target)))
						missing-outer-key-repr-map)
					(if (nil? internal-target)
						domain-key-local-repr-map
						(target-safe-repr-map
							domain-key-local-repr-map internal-target)))))
		(define nested-domain-tables
			(if internal-target-ordered-limit-one
				(qpu-low-domainize-target-source
					merged-tables internal-target
					domain-left-tuple right-tuple nested-target-domain-pred)
				merged-tables))
		(define domain-tables
			(if (nil? internal-target)
				nested-domain-tables
				(qpu-low-add-join-cond-to-table
					nested-domain-tables internal-target
					internal-target-nonlocal-cond-effective)))
		(define internal-target-outer?
			(and (not (nil? internal-target))
				(qpu-low-table-entry-is-outer?
					domain-tables internal-target)))
		(define nonouter-join-cond
			(qpu-low-nonouter-table-join-cond domain-tables))
		(define domain-key-not-nil-cond
			(qpu-low-filter-cond-by-aliases
				(qpu-low-domain-key-not-nil-cond domain-fields-final-base)
				domain-aliases))
		(define effective-value-pred
			(qpu-low-strip-tautological-equalities
				(qpu-low-and-cond
					internal-pred-effective nonouter-join-cond)))
		(define effective-value-helper-from-pred
			(qpu-low-first-scalar-helper-ref-alias effective-value-pred))
		(define effective-value-helper
			(coalesce effective-value-helper-from-pred
				(qpu-low-first-scalar-helper-ref-alias
					domain-fields-final-base)))
		(define projected-value-helper
			(qpu-low-first-scalar-helper-ref-alias
				(qpp-tuple-fields right-tuple)))
		(define domain-table-value-helper
			(reduce (coalesceNil domain-tables '()) (lambda (found td) (match td
				'(alias_ _ source _ _)
				(if (not (nil? found)) found
					(if (and
						(qpu-low-scalar-helper-alias? alias_)
						(or
							(not (qpp-tuple? source))
							(qpu-low-fields-has-name
								(qpp-fields-to-pairs (qpp-tuple-fields source))
								"value")))
						alias_
						nil))
				found))
				nil))
		(define effective-value-helper-for-payload
			(coalesce effective-value-helper
				projected-value-helper
				domain-table-value-helper))
		(define effective-value-helper-ordered-limit-one
			(and (not (nil? effective-value-helper))
				(qpu-low-target-source-ordered-limit-one?
					domain-tables effective-value-helper)))
		(define effective-value-pred-count-helper?
			(or
				(and
					(qpu-low-predicate-is-not? internal-pred)
					(or
						(not (nil? effective-value-helper-from-pred))
						(not (nil? internal-target))))
				(reduce (qpu-and-conjuncts effective-value-pred)
					(lambda (found part)
						(or found
							(qpu-low-count-helper-value-predicate? part)))
					false)))
			(define anti-count-helper-alias
				(coalesce
					(qpu-low-anti-count-helper-alias-in-expr internal-pred)
					(qpu-low-anti-count-helper-alias-in-expr effective-value-pred)
					(qpu-low-anti-count-helper-alias-in-fields right-fields)
					(qpu-low-anti-count-helper-alias-in-fields
						domain-fields-final-base)
					(if (qpu-low-predicate-is-not? internal-pred)
						(coalesce effective-value-helper-from-pred internal-target)
						nil)))
		(define effective-helper-key-fields
			(if (or (nil? effective-value-helper)
				effective-value-pred-count-helper?
				effective-value-helper-ordered-limit-one) '()
				(reduce (coalesceNil domain-tables '()) (lambda (acc td) (match td
					'(alias_ _ source _ _)
					(if (and (equal? alias_ effective-value-helper)
						(qpp-tuple? source))
						(reduce (qpp-fields-to-pairs (qpp-tuple-fields source)) (lambda (acc2 pair) (match pair
							'(field-name _)
							(if (or
								(nil? field-name)
								(list? field-name)
								(equal?? field-name "value"))
								acc2
								(begin
									(define field-name-str (string field-name))
									(define source-col (qpu-low-kt-ref-source-col field-name-str))
									(define key-field-name (if (and
										(>= (strlen field-name-str) 5)
										(equal? (substr field-name-str 0 5) "__kt_"))
										field-name-str
										(concat "__kt_" source-col)))
									(if (qpu-low-fields-has-name
										(merge domain-fields-final-base acc2)
										key-field-name)
										acc2
										(merge acc2 (list
											(list key-field-name
												(list (quote get_column)
													effective-value-helper false field-name false)))))))
							acc2))
							acc)
						acc)
					acc))
					'())))
		(define effective-helper-value-fields
			(if (or (nil? effective-value-helper-for-payload)
				(qpu-low-fields-has-name
					domain-fields-final-base "value"))
				'()
				(reduce (coalesceNil domain-tables '()) (lambda (acc td) (match td
					'(alias_ _ source _ _)
					(if (and (equal? alias_ effective-value-helper-for-payload)
						(qpp-tuple? source)
						(qpu-low-fields-has-name
							(qpp-fields-to-pairs (qpp-tuple-fields source))
							"value"))
						(list
							(list "value"
								(list (quote get_column)
									effective-value-helper-for-payload false "value" false)))
						acc)
					acc))
					'())))
		(define domain-fields-final-effective-base
			(merge domain-fields-final-base
				effective-helper-key-fields
				effective-helper-value-fields))
		(define domain-fields-final-effective
			(if (or
				(not effective-value-pred-count-helper?)
				(nil? anti-count-helper-alias))
				domain-fields-final-effective-base
				(filter domain-fields-final-effective-base
					(lambda (pair) (match pair
						'(name expr)
						(not
							(and
								(not (equal?? name "value"))
								(qpu-low-expr-refs-alias?
									expr anti-count-helper-alias)))
						true)))))
		(define effective-value-pred-is-rhs-ordered-limit-one
			(and effective-value-helper-ordered-limit-one
				(equal? effective-value-helper rhs-alias)))
		(define anti-count-domain?
			(and
				top-down-grouped-domain?
				(not (nil? anti-count-helper-alias))))
		(define conditioned-domain-fields-final
			(if (or (nil? effective-value-pred)
				(equal? effective-value-pred true)
				(equal? effective-value-pred (quote true))
				effective-value-pred-is-rhs-ordered-limit-one)
				domain-fields-final-effective
				(qpu-low-wrap-domain-value-fields-for-domain
					domain-fields-final-effective effective-value-pred)))
		(define collapse-limit-one-final
			(qpu-low-domain-collapse-limit-one?
				right-tuple conditioned-domain-fields-final))
		(define output-fields-final-raw
			(if collapse-limit-one-final
				(qpu-low-collapse-domain-fields
					conditioned-domain-fields-final)
				conditioned-domain-fields-final))
		(define scalar-helper-value-defaults-false? (lambda (alias_)
			(reduce (coalesceNil domain-tables '()) (lambda (found td) (match td
				'(a _ source _ _)
				(or found
					(and
						(equal? a alias_)
						(qpp-tuple? source)
						(begin
							(define source-value
								(qpu-low-field-expr-by-name
									(qpp-tuple-fields source) "value"))
							(qpu-low-exists-helper-value-output?
								source-value))))
				found))
				false)))
		(define expr-uses-default-false-scalar-helper-value? (lambda (expr)
			(match expr
				'((symbol get_column) alias_ _ col _)
				(and
					(equal?? col "value")
					(scalar-helper-value-defaults-false? alias_))
				'((quote get_column) alias_ _ col _)
				(and
					(equal?? col "value")
					(scalar-helper-value-defaults-false? alias_))
				(cons head args)
				(or
					(expr-uses-default-false-scalar-helper-value? head)
					(reduce (coalesceNil args '()) (lambda (found arg)
						(or found
							(expr-uses-default-false-scalar-helper-value?
								arg)))
						false))
				false)))
		(define default-missing-scalar-helper-value (lambda (expr)
			(match expr
				'((symbol get_column) alias_ _ col _)
				(if (and
					(equal?? col "value")
					(scalar-helper-value-defaults-false? alias_))
					(list (quote coalesceNil) expr false)
					expr)
				'((quote get_column) alias_ _ col _)
				(if (and
					(equal?? col "value")
					(scalar-helper-value-defaults-false? alias_))
					(list (quote coalesceNil) expr false)
					expr)
				'(head inner reducer neutral)
				(if (and
					(qpu-low-head-is? head (quote aggregate))
					(expr-uses-default-false-scalar-helper-value?
						inner))
					(list head inner reducer false)
					expr)
				expr)))
		(define output-fields-final
			(map output-fields-final-raw (lambda (pair) (match pair
				'(name expr)
				(list name
					(if (equal?? name "value")
						(default-missing-scalar-helper-value expr)
						expr))
				pair))))
		(define output-group-final
			(if collapse-limit-one-final
				(qpu-low-domain-key-group
					conditioned-domain-fields-final)
				(qpu-low-rewrite-domain-key-list
					(qpp-tuple-group right-tuple)
					right-fields domain-key-map)))
		(define inner-domain-cond
			(qpu-low-strip-tautological-equalities
				(qpu-low-and-cond
					(if (nil? internal-target)
						(qpu-low-and-cond
							(qpp-tuple-condition right-tuple)
							internal-pred)
						(if (qpu-low-predicate-is-not? internal-pred)
							(qpp-tuple-condition right-tuple)
							(qpu-low-and-cond
								(qpp-tuple-condition right-tuple)
								(qpu-low-and-cond
									internal-pred-effective
									(if internal-target-outer?
										nil
										internal-target-nonlocal-cond-effective)))))
					nonouter-join-cond)))
		(define inner-domain-cond-for-where
			(if top-down-grouped-domain?
				(qpp-tuple-condition right-tuple)
				inner-domain-cond))
		(define grouped-domain-probe-alias
			(if (not top-down-grouped-domain?) nil
				(begin
					(define left-aliases-for-domain
						(qpu-low-table-aliases
							(qpp-tuple-tables domain-left-tuple)))
					(reduce (qpu-low-table-aliases
						(qpp-tuple-tables right-tuple)) (lambda (found alias_)
							(if (not (nil? found)) found
								(if (has? left-aliases-for-domain alias_) nil alias_)))
						nil))))
			(define grouped-domain-mark-probe-outer (lambda (tables)
				(if (nil? grouped-domain-probe-alias)
					tables
					(map (coalesceNil tables '()) (lambda (td) (match td
						'(a s t io je)
						(if (equal? a grouped-domain-probe-alias)
							(list a s t true (qpu-low-and-cond je true))
							td)
						td))))))
			(define final-domain-target
				(if anti-count-domain?
					nil
					internal-target))
			(define final-domain-tables-base
				(qpu-low-prioritize-table-alias
					(grouped-domain-mark-probe-outer
						(if effective-value-pred-is-rhs-ordered-limit-one
							(qpu-low-add-join-cond-to-table
								domain-tables effective-value-helper effective-value-pred)
							domain-tables))
					final-domain-target))
			(define final-domain-tables-prioritized
				(if anti-count-domain?
					final-domain-tables-base
					(qpu-low-prioritize-scalar-helper-tables
						final-domain-tables-base)))
			(define final-domain-tables
				(qpu-low-order-tables-by-join-deps
					(if anti-count-domain?
						final-domain-tables-prioritized
						(qpu-low-move-scalar-helper-join-conds
							final-domain-tables-prioritized))))
		(define final-domain-aliases
			(qpu-low-table-aliases final-domain-tables))
		(define final-domain-local-helper-key-repr (lambda (expr)
			(begin
				(define info
					(coalesce
						(qpu-low-col-ref-info expr)
						(qpu-low-dot-ref-info expr)
						(qpu-low-outer-dot-ref-info expr)))
				(if (or (nil? info) (has? final-domain-aliases (nth info 0)))
					nil
					(reduce (coalesceNil output-fields-final '()) (lambda (found pair) (match pair
						'(name repl)
						(if (not (nil? found))
							found
							(if (and
								(qpu-low-domain-key-field? pair)
								(equal? (qpu-low-kt-ref-source-col name) (nth info 1))
								(not (qpu-low-expr-has-ref-outside? repl final-domain-aliases))
								(qpu-low-expr-has-scalar-helper-ref? repl))
								repl
								nil))
						found))
						nil)))))
			(define final-domain-local-mapped-key-repr (lambda (expr)
				(begin
					(define info
						(coalesce
							(qpu-low-col-ref-info expr)
							(qpu-low-dot-ref-info expr)
							(qpu-low-outer-dot-ref-info expr)))
					(define outer-expr
						(qpu-low-outer-dot-ref-from-info info))
					(define key-name
						(reduce (coalesceNil domain-key-map '()) (lambda (found pair) (match pair
							'(name key-expr)
							(if (not (nil? found))
							found
							(if (or (equal? key-expr expr)
								(and (not (nil? outer-expr))
									(equal? key-expr outer-expr)))
								name
									nil))
							found))
							nil))
					(if (nil? key-name)
						nil
						(reduce (coalesceNil output-fields-final '()) (lambda (found pair) (match pair
							'(name repl)
							(if (not (nil? found))
								found
							(if (and
								(equal? name key-name)
								(not (qpu-low-expr-has-ref-outside? repl final-domain-aliases)))
									repl
									nil))
							found))
							nil)))))
		(define final-domain-left-cond
			(qpu-low-filter-cond-by-aliases
				(qpp-tuple-condition domain-left-tuple)
				final-domain-aliases))
		(define final-domain-cond
			(qpu-and-from-conjuncts
				(filter
					(qpu-and-conjuncts
						(qpu-low-filter-cond-by-aliases
							(qpu-low-and-cond
								(qpu-low-and-cond
									final-domain-left-cond
									inner-domain-cond-for-where)
								domain-key-not-nil-cond)
							final-domain-aliases))
					(lambda (conj)
						(and
							(or
								(not top-down-grouped-domain?)
								(not (reduce
									(qpu-low-expr-ref-aliases conj)
									(lambda (found alias_)
										(or found
											(qpu-low-scalar-helper-alias?
												alias_)))
									false)))
							(not (qpu-low-expr-refs-alias?
								conj rhs-alias))
							(not (qpu-low-expr-has-explicit-outer? conj))
							(not (qpu-low-expr-has-ref-outside?
								conj final-domain-aliases)))))))
		(define final-domain-nonhelper-equality-repr (lambda (expr)
			(reduce (coalesceNil final-domain-tables '()) (lambda (found td) (match td
				'(_ _ _ _ je)
				(if (not (nil? found))
					found
					(reduce (qpu-and-conjuncts (coalesceNil je true)) (lambda (found2 conj) (match conj
						'(op lhs rhs)
						(if (and (nil? found2) (qpu-low-equality-op? op))
							(begin
								(define candidate
									(if (equal? lhs expr) rhs
										(if (equal? rhs expr) lhs nil)))
								(if (and
									(not (nil? candidate))
									(not (qpu-low-expr-has-scalar-helper-ref?
										candidate))
									(not (qpu-low-expr-has-ref-outside?
										candidate final-domain-aliases)))
									candidate
									nil))
							found2)
						found2))
						nil))
				found))
				nil)))
		(define final-domain-wrap-expr (lambda (expr)
			(coalesce
				(if (and
					top-down-grouped-domain?
					right-projects-count-helper-value?
					(qpu-low-expr-has-scalar-helper-ref? expr))
					(final-domain-nonhelper-equality-repr expr)
					nil)
				(final-domain-local-mapped-key-repr expr)
				(final-domain-local-helper-key-repr expr)
				(qpu-low-add-outer-hop-for-nonlocal-refs
					expr final-domain-aliases))))
		(define final-domain-output-aliases
			(qpu-low-table-aliases-recursive final-domain-tables))
		(define final-domain-wrap-output-plain-expr (lambda (expr)
			(coalesce
				(if (and
					top-down-grouped-domain?
					right-projects-count-helper-value?
					(qpu-low-expr-has-scalar-helper-ref? expr))
					(final-domain-nonhelper-equality-repr expr)
					nil)
				(final-domain-local-mapped-key-repr expr)
				(final-domain-local-helper-key-repr expr)
				(qpu-low-add-outer-hop-for-nonlocal-refs
					expr final-domain-output-aliases))))
		(define derived-scalar-domain-alias? (lambda (alias_)
			(reduce (coalesceNil final-domain-tables '()) (lambda (found td)
				(or found
					(match td
						'(a _ source _ _)
						(and
							(equal? a alias_)
							(qpu-low-scalar-helper-alias? a)
							(qpp-tuple? source))
						false)))
				false)))
			(define executable-domain-cond (lambda (cond)
				(qpu-and-from-conjuncts
					(filter (qpu-and-conjuncts cond)
						(lambda (conj)
							(and
								(not (and
									(qpu-low-expr-has-explicit-outer? conj)
									(reduce
										(qpu-low-expr-ref-aliases conj)
										(lambda (found alias_)
											(or found
												(qpu-low-scalar-helper-alias? alias_)))
										false)))
								(not (reduce
									(qpu-low-expr-ref-aliases conj)
									(lambda (found alias_)
										(or found
											(and
												(qpu-low-scalar-helper-alias? alias_)
												(or
													(derived-scalar-domain-alias? alias_)
													(not (has? final-domain-aliases alias_))))))
									false))))))))
			(define executable-domain-tables
				(map (coalesceNil final-domain-tables '()) (lambda (td) (match td
					'(a s t io je)
					(list a s t io
						(executable-domain-cond je))
					td))))
			(define final-domain-output-local-pred (lambda (pred)
				(executable-domain-cond
					(qpu-and-from-conjuncts
						(filter (qpu-and-conjuncts pred)
							(lambda (conj)
								(and
									(not (qpu-low-expr-has-explicit-outer? conj))
									(not (qpu-low-expr-has-ref-outside?
										conj final-domain-aliases)))))))))
			(define final-domain-wrap-output-expr (lambda (expr)
				(match expr
					'(head pred then else)
					(if (qpu-low-head-is? head (quote if))
						(begin
							(define local-pred
								(final-domain-output-local-pred pred))
							(define wrapped-then
								(final-domain-wrap-output-expr then))
							(if (or (nil? local-pred)
								(equal? local-pred true)
								(equal? local-pred (quote true)))
									wrapped-then
									(list head
										(final-domain-wrap-expr local-pred)
										wrapped-then
										(final-domain-wrap-output-expr else))))
						(final-domain-wrap-output-plain-expr expr))
					(final-domain-wrap-output-plain-expr expr))))
		(qpu-low-tag-domain-ordered-limit-one
			(qpp-rebuild-tuple
				(qpp-tuple-schema right-tuple)
				executable-domain-tables
				(map (coalesceNil output-fields-final '()) (lambda (pair) (match pair
					'(name expr) (list name (final-domain-wrap-output-expr expr))
					pair)))
				(final-domain-wrap-expr
					(executable-domain-cond final-domain-cond))
				(reduce
					(map (coalesceNil output-group-final '())
						final-domain-wrap-expr)
					(lambda (acc expr)
						(if (has? acc expr) acc (merge acc (list expr))))
					'())
				(if (nil? (qpp-tuple-having right-tuple))
					nil
					(final-domain-wrap-expr (qpp-tuple-having right-tuple)))
				(map (coalesceNil (qpp-tuple-order right-tuple) '()) (lambda (item) (match item
					'(expr dir) (list (final-domain-wrap-expr expr) dir)
					item)))
				(if collapse-limit-one-final nil (qpp-tuple-limit right-tuple))
				(if collapse-limit-one-final nil (qpp-tuple-offset right-tuple)))
			output-fields-final)))))

(define qpu-low-null-tolerate-nested-scalar-key-conjunct (lambda (conj rhs-alias fields)
	conj))

(define qpu-low-nested-scalar-key-replacement (lambda (expr rhs-alias fields)
	(match expr
		'((symbol get_column) tv _ col _)
		(if (equal? tv rhs-alias)
			(begin
				(define fexpr (qpu-low-field-expr-by-name fields col))
				(if (and (not (nil? fexpr))
					(qpu-low-expr-has-scalar-helper-ref? fexpr))
					fexpr
					nil))
			nil)
		'((quote get_column) tv _ col _)
		(if (equal? tv rhs-alias)
			(begin
				(define fexpr (qpu-low-field-expr-by-name fields col))
				(if (and (not (nil? fexpr))
					(qpu-low-expr-has-scalar-helper-ref? fexpr))
					fexpr
					nil))
			nil)
		nil)))

(define qpu-low-rewrite-nested-scalar-key-conjunct (lambda (conj rhs-alias fields)
	(match conj
		'(op lhs rhs)
		(begin
			(define lhs-repl (qpu-low-nested-scalar-key-replacement lhs rhs-alias fields))
			(define rhs-repl (qpu-low-nested-scalar-key-replacement rhs rhs-alias fields))
			(if (not (nil? lhs-repl))
				(list op lhs-repl rhs)
				(if (not (nil? rhs-repl))
					(list op lhs rhs-repl)
					nil)))
		nil)))

(define qpu-low-first-scalar-helper-ref-alias (lambda (expr)
	(match expr
		'(get_column tv _ _ _)
		(if (qpu-low-scalar-helper-alias? tv) tv nil)
		'((symbol get_column) tv _ _ _)
		(if (qpu-low-scalar-helper-alias? tv) tv nil)
		'((quote get_column) tv _ _ _)
		(if (qpu-low-scalar-helper-alias? tv) tv nil)
		(cons head args)
		(coalesce
			(qpu-low-first-scalar-helper-ref-alias head)
			(reduce (coalesceNil args '()) (lambda (found a)
				(if (not (nil? found)) found
					(qpu-low-first-scalar-helper-ref-alias a))) nil))
		nil)))

(define qpu-low-table-subtuple-by-alias (lambda (tables alias_)
	(reduce (coalesceNil tables '()) (lambda (found td)
		(if (not (nil? found)) found
			(match td
				'(a _ t _ _)
				(if (and (equal? a alias_) (qpp-tuple? t)) t nil)
				nil)))
		nil)))

(define qpu-low-expand-scalar-helper-field-refs (lambda (expr tables)
	(match expr
		'((symbol get_column) tv _ col _)
		(if (qpu-low-scalar-helper-alias? tv)
			(begin
				(define sub (qpu-low-table-subtuple-by-alias tables tv))
				(define fexpr (if (nil? sub) nil
					(qpu-low-field-expr-by-name (qpp-tuple-fields sub) col)))
				(if (nil? fexpr) expr
					(qpu-low-expand-scalar-helper-field-refs
						fexpr (qpp-tuple-tables sub))))
			expr)
		'((quote get_column) tv _ col _)
		(if (qpu-low-scalar-helper-alias? tv)
			(begin
				(define sub (qpu-low-table-subtuple-by-alias tables tv))
				(define fexpr (if (nil? sub) nil
					(qpu-low-field-expr-by-name (qpp-tuple-fields sub) col)))
				(if (nil? fexpr) expr
					(qpu-low-expand-scalar-helper-field-refs
						fexpr (qpp-tuple-tables sub))))
			expr)
		(cons head args)
		(cons head (map (coalesceNil args '())
			(lambda (a) (qpu-low-expand-scalar-helper-field-refs a tables))))
		expr)))

(define qpu-low-add-join-cond-to-table (lambda (tables target cond)
	(map (coalesceNil tables '()) (lambda (td) (match td
		'(a s t io je)
		(if (equal? a target)
			(list a s t io (qpu-low-and-cond je cond))
			(if (qpp-tuple? t)
				(begin
					(define nested-local-aliases
						(map (coalesceNil (qpp-tuple-tables t) '()) (lambda (ntd)
							(match ntd
								'(na _ _ _ _) na
								nil))))
					(define nested-cond
						(qpu-low-add-outer-hop-for-nonlocal-refs
							cond nested-local-aliases))
					(list a s
						(qpp-rebuild-tuple
							(qpp-tuple-schema t)
							(qpu-low-add-join-cond-to-table
								(qpp-tuple-tables t) target nested-cond)
							(qpp-tuple-fields t)
							(qpp-tuple-condition t)
							(qpp-tuple-group t)
							(qpp-tuple-having t)
							(qpp-tuple-order t)
							(qpp-tuple-limit t)
							(qpp-tuple-offset t))
						io je))
				td))
		td)))))

(define qpu-low-hoist-nested-scalar-key-conjuncts (lambda (tuple rhs-alias pred)
	(begin
		(define fields (qpp-tuple-fields tuple))
		(define step (lambda (acc conj)
			(begin
				(define nested
					(qpu-low-rewrite-nested-scalar-key-conjunct
						conj rhs-alias fields))
				(if (nil? nested)
					(list (nth acc 0) (merge (nth acc 1) (list conj)))
					(begin
						(define target
							(qpu-low-first-scalar-helper-ref-alias nested))
						(if (nil? target)
							(list (nth acc 0) (merge (nth acc 1) (list conj)))
							(begin
								(list
									(qpu-low-add-join-cond-to-table
										(nth acc 0) target nested)
									(nth acc 1)))))))))
		(define result
			(reduce (qpu-and-conjuncts pred) step
				(list (qpp-tuple-tables tuple) '())))
		(list
			(qpp-rebuild-tuple
				(qpp-tuple-schema tuple)
				(nth result 0)
				(qpp-tuple-fields tuple)
				(qpp-tuple-condition tuple)
				(qpp-tuple-group tuple)
				(qpp-tuple-having tuple)
				(qpp-tuple-order tuple)
				(qpp-tuple-limit tuple)
				(qpp-tuple-offset tuple))
			(qpu-and-from-conjuncts (nth result 1))))))

/* qpu-low-inline-scalar-eligible? — true when the scalar dep-join's right
side can be inlined as flat (instead of derived-wrap). Conservative gate
matching qpu-low-tag-inner-once-limit:
- join-type=left
- rhs-alias is a scalar-helper alias
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
		(define right-has-nested-scalar-helper-ref
			(qpu-low-expr-has-scalar-helper-ref?
				(list
					(qpp-tuple-fields right-tuple)
					(qpp-tuple-condition right-tuple)
					(qpp-tuple-group right-tuple)
					(qpp-tuple-having right-tuple)
					(qpp-tuple-order right-tuple))))
		(if (not (and
			(equal? join-type (quote left))
			(qpu-low-scalar-helper-alias? rhs-alias)
			(equal? (qpp-tuple-limit right-tuple) 1)
			(equal? (count tbls) 1)
			(equal? (count flds) 1)
			(equal? (count grp) 0)
			(nil? hav)
			(not (qpu-low-expr-has-scalar-helper-ref? join-pred))
			(not right-has-nested-scalar-helper-ref)
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
					false))))))))

/* qpu-low-join-inline-scalar — produce a flat-join outer tuple instead of
wrapping right as derived. Inline the inner table with scan-tagged-table
carrying correlation cols at front of sortcols and partition_cols=N.
Outer field refs to helper.value get rewritten to inner-aliased col directly.

Aliasing: the inner table is REALIASED to rhs-alias (a scalar-helper alias) so multiple
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
		(define inline-null-sensitive-predicate? (lambda (expr) (match expr
			'((symbol coalesce) _ _) true
			'((quote coalesce) _ _) true
			'((symbol coalesceNil) _ _) true
			'((quote coalesceNil) _ _) true
			(cons head args) (or (inline-null-sensitive-predicate? head)
				(reduce (coalesceNil args '()) (lambda (found arg)
					(or found (inline-null-sensitive-predicate? arg))) false))
			false)))
		(define inner-where-stays-above
			(and (not (nil? inner-where-retargeted))
				(inline-null-sensitive-predicate? inner-where-retargeted)))
		(define inner-where-for-join
			(if inner-where-stays-above nil inner-where-retargeted))
		(define inner-where-residual
			(if inner-where-stays-above inner-where-retargeted nil))
		(define join-pred-retargeted (qpu-low-rewrite-refs
			join-pred retarget-aliases rhs-alias))
		(define join-pred-for-join
			(if (equal? join-type (quote left))
				(qpu-and-from-conjuncts
					(filter (qpu-and-conjuncts join-pred-retargeted)
						qpu-low-equality-conjunct?))
				join-pred-retargeted))
		(define local-inline-aliases (merge left-aliases (list rhs-alias)))
		(define wrap-nonlocal-inline-refs (lambda (expr) (match expr
			'((symbol outer) _) expr
			'((quote outer) _) expr
			'((symbol get_column) tv ti col ci)
			(if (has? local-inline-aliases tv)
				expr
				(list (quote outer) (list (quote outer) (symbol (concat tv "." col)))))
			'((quote get_column) tv ti col ci)
			(if (has? local-inline-aliases tv)
				expr
				(list (quote outer) (list (quote outer) (symbol (concat tv "." col)))))
			(cons head args)
			(cons head (map (coalesceNil args '()) wrap-nonlocal-inline-refs))
			expr)))
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
		(define combined-je (wrap-nonlocal-inline-refs
			(qpu-low-and-cond join-pred-for-join inner-where-for-join)))
		(define new-entry
			(list rhs-alias inner-schema tagged-tname true combined-je))
		(define normalize-inline-order (lambda (ord alias_)
			(map (coalesceNil ord '()) (lambda (item) (match item
				'(expr dir)
				(list (qpu-low-rewrite-refs expr (list alias_) "__inline") dir)
				item)))))
		(define normalize-inline-expr (lambda (expr alias_)
			(qpu-low-rewrite-refs expr (list alias_) "__inline")))
		(define reusable-inline-alias
			(if (not (nil? inner-where-residual))
				nil
				(reduce (coalesceNil (qpp-tuple-tables left-tuple) '()) (lambda (found ltd)
					(if (not (nil? found))
						found
						(match ltd
							'(left-alias left-schema left-source left-outer left-je)
							(if (and
								left-outer
								(equal? left-schema inner-schema)
								(qpu-low-scan-tagged-table? left-source)
								(equal? (scan_tagged_table_base left-source) inner-base)
								(equal? (scan_tagged_table_limit left-source)
									(scan_tagged_table_limit tagged-tname))
								(equal? (scan_tagged_table_offset left-source)
									(scan_tagged_table_offset tagged-tname))
								(equal? (scan_tagged_table_partition_cols left-source)
									(scan_tagged_table_partition_cols tagged-tname))
								(equal? (scan_tagged_table_once_limit left-source)
									(scan_tagged_table_once_limit tagged-tname))
								(equal?
									(normalize-inline-order
										(scan_tagged_table_order left-source) left-alias)
									(normalize-inline-order
										(scan_tagged_table_order tagged-tname) rhs-alias))
								(equal?
									(normalize-inline-expr left-je left-alias)
									(normalize-inline-expr combined-je rhs-alias)))
									left-alias
									nil)
									nil)))
							nil)))
		(define effective-field-expr
			(if (nil? reusable-inline-alias)
				field-expr
				(qpu-low-rewrite-refs
					field-expr-raw retarget-aliases reusable-inline-alias)))
		/* Register the (rhs-alias . field-name) → field-expr rewrite so any
		OUTER wrapper (qpir-select, qpir-map, qpir-groupby) above us picks
		it up when applying its own predicates/projections. Lift placed
		scalar-helper field refs in the outer's WHERE/SELECT — those scopes are
		processed at a different lowering step than ours. */
		(qpu-low-sq-rewrites-add rhs-alias field-name effective-field-expr)
		/* (rewrite map populated for any future wrapper that needs it —
		currently only used by the local rewrite below; the qpu-low-select
		integration is deferred since it caused esv test regressions —
		trades 1 esv win for 4 esv losses.) */
		(define rewritten-fields
			(qpu-low-replace-sq-field (qpp-tuple-fields left-tuple)
				rhs-alias field-name effective-field-expr))
		(define rewritten-cond
			(qpu-low-replace-sq-field-expr (qpp-tuple-condition left-tuple)
				rhs-alias field-name effective-field-expr))
		(define rewritten-having
			(if (nil? (qpp-tuple-having left-tuple)) nil
				(qpu-low-replace-sq-field-expr (qpp-tuple-having left-tuple)
					rhs-alias field-name effective-field-expr)))
		(define rewritten-group
			(map (coalesceNil (qpp-tuple-group left-tuple) '())
				(lambda (gexpr)
					(qpu-low-replace-sq-field-expr
						gexpr rhs-alias field-name effective-field-expr))))
		(define rewritten-order
			(map (coalesceNil (qpp-tuple-order left-tuple) '())
				(lambda (order-item) (match order-item
					'(order-expr order-dir)
					(list (qpu-low-replace-sq-field-expr
							order-expr rhs-alias field-name effective-field-expr)
						order-dir)
					order-item))))
		(define inline-result
			(qpp-rebuild-tuple
				(qpp-tuple-schema left-tuple)
				(if (nil? reusable-inline-alias)
					(merge
						(qpu-low-carry-left-tables-one-scope-deeper
							(qpp-tuple-tables left-tuple)
							local-inline-aliases)
						(list new-entry))
					(qpp-tuple-tables left-tuple))
				rewritten-fields
				(qpu-low-and-cond rewritten-cond inner-where-residual)
				rewritten-group
				rewritten-having
				rewritten-order
				(qpp-tuple-limit left-tuple)
				(qpp-tuple-offset left-tuple)))
		inline-result)))

(define qpu-low-rewrite-outer-dot-refs (lambda (expr from-aliases to-alias)
	(match expr
		'((symbol outer) symname)
		(if (list? symname)
			(list (quote outer) (qpu-low-rewrite-outer-dot-refs symname from-aliases to-alias))
			(match (split (string symname) ".")
				'(tv col) (if (has? from-aliases tv)
					(list (quote outer) (symbol (concat to-alias "." col)))
					expr)
				_ expr))
		'((quote outer) symname)
		(if (list? symname)
			(list (quote outer) (qpu-low-rewrite-outer-dot-refs symname from-aliases to-alias))
			(match (split (string symname) ".")
				'(tv col) (if (has? from-aliases tv)
					(list (quote outer) (symbol (concat to-alias "." col)))
					expr)
				_ expr))
		(cons head args)
		(cons (qpu-low-rewrite-outer-dot-refs head from-aliases to-alias)
			(map (coalesceNil args '()) (lambda (a)
				(qpu-low-rewrite-outer-dot-refs a from-aliases to-alias))))
		expr)))

(define qpu-low-retarget-table-entry (lambda (td aliases rhs-alias)
	(match td
		'(a s t io je)
		(list a s t io
			(if (nil? je) nil
				(qpu-low-rewrite-outer-dot-refs
					(qpu-low-rewrite-refs je aliases rhs-alias)
					aliases rhs-alias)))
		td)))

(define qpu-low-join-inline-scalar-pipeline (lambda (left-tuple right-tuple join-pred rhs-alias join-type)
	(begin
		(define tbls (qpp-tuple-tables right-tuple))
		(define first-td (nth tbls 0))
		(define inner-natural-alias (nth first-td 0))
		(define inner-schema (nth first-td 1))
		(define inner-base (nth first-td 2))
		(define inner-join (nth first-td 4))
		(define inner-cond (qpp-tuple-condition right-tuple))
		(define inner-order (coalesceNil (qpp-tuple-order right-tuple) '()))
		(define inner-offset (qpp-tuple-offset right-tuple))
		(define retarget-aliases (list inner-natural-alias))
		(define pipeline-left-aliases (map (qpp-tuple-tables left-tuple) (lambda (t)
			(if (or (nil? t) (< (count t) 1)) nil (nth t 0)))))
		(define pipeline-corr-cols
			(qpu-low-corr-from-pred join-pred inner-natural-alias pipeline-left-aliases))
		(define pipeline-corr-order
			(map pipeline-corr-cols (lambda (c) (list
				(list (quote get_column) rhs-alias false c false)
				(quote <)))))
		(define first-source
			(if (and
				(equal? (qpp-tuple-limit right-tuple) 1)
				(equal? (count inner-order) 0)
				(> (count pipeline-corr-cols) 0)
				(not (qpp-tuple? inner-base))
				(not (qpu-low-scan-tagged-table? inner-base)))
				(make_scan_tagged_table
					inner-base pipeline-corr-order 1 inner-offset
					(count pipeline-corr-cols) 1)
				inner-base))
		(define pipeline-join-pred-retargeted
			(qpu-low-rewrite-refs join-pred retarget-aliases rhs-alias))
		(define pipeline-join-pred-for-join
			(if (equal? join-type (quote left))
				(qpu-and-from-conjuncts
					(filter (qpu-and-conjuncts pipeline-join-pred-retargeted)
						qpu-low-equality-conjunct?))
				pipeline-join-pred-retargeted))
		(define combined-first-je
			(qpu-low-and-cond
				pipeline-join-pred-for-join
				(qpu-low-and-cond
					(if (nil? inner-join) nil
						(qpu-low-rewrite-refs inner-join retarget-aliases rhs-alias))
					(if (or (nil? inner-cond) (equal? inner-cond true)
						(equal? inner-cond (quote true)))
						nil
						(qpu-low-rewrite-refs inner-cond retarget-aliases rhs-alias)))))
		(define first-entry (list rhs-alias inner-schema first-source true combined-first-je))
		(define retargeted-rest-entries
			(map (cdr tbls) (lambda (td)
				(qpu-low-retarget-table-entry td retarget-aliases rhs-alias))))
		(define pipeline-local-aliases
			(merge (list rhs-alias)
				(map retargeted-rest-entries (lambda (td) (match td
					'(a _ _ _ _) a
					nil)))))
		(define rest-aliases
			(map retargeted-rest-entries (lambda (td) (match td
				'(a _ _ _ _) a
				nil))))
		(define rest-entry-depends-on-rest? (lambda (td) (match td
			'(a _ _ _ je)
			(and (not (nil? je))
				(reduce (extract_tblvars je) (lambda (found tv)
					(or found (and (not (equal? tv a)) (has? rest-aliases tv))))
					false))
			false)))
		(define rest-entries
			(map retargeted-rest-entries (lambda (td)
				(if (rest-entry-depends-on-rest? td)
					(car (qpu-low-carry-pipeline-rest-one-scope-deeper
						(list td)
						pipeline-local-aliases))
					td))))
		(define fpair (nth (qpp-fields-to-pairs (qpp-tuple-fields right-tuple)) 0))
		(define field-name (nth fpair 0))
		(define field-expr
			(qpu-low-rewrite-refs (nth fpair 1) retarget-aliases rhs-alias))
		(qpu-low-sq-rewrites-add rhs-alias field-name field-expr)
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
		(define rewritten-group
			(map (coalesceNil (qpp-tuple-group left-tuple) '())
				(lambda (gexpr)
					(qpu-low-replace-sq-field-expr gexpr rhs-alias field-name field-expr))))
		(define rewritten-order
			(map (coalesceNil (qpp-tuple-order left-tuple) '())
				(lambda (order-item) (match order-item
					'(order-expr order-dir)
					(list (qpu-low-replace-sq-field-expr order-expr rhs-alias field-name field-expr)
						order-dir)
					order-item))))
		(qpu-low-push-single-alias-condition-to-tables
			(qpp-rebuild-tuple
				(qpp-tuple-schema left-tuple)
				(merge (qpp-tuple-tables left-tuple)
					(cons first-entry rest-entries))
				rewritten-fields
				rewritten-cond
				rewritten-group
				rewritten-having
				rewritten-order
				(qpp-tuple-limit left-tuple)
				(qpp-tuple-offset left-tuple))))))

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

(define qpu-low-domain-key-field-name? (lambda (name)
	(and
		(not (nil? name))
		(not (list? name))
		(begin
			(define name-str (string name))
			(and
				(>= (strlen name-str) 5)
				(equal? (substr name-str 0 5) "__kt_"))))))

(define qpu-low-scalar-domain-normalize-expr (lambda (expr)
	(match expr
		'((symbol get_column) tv ti col ci)
		(list (quote get_column)
			(if (qpu-low-scalar-helper-alias? tv) "__scalar_domain" tv)
			ti col ci)
		'((quote get_column) tv ti col ci)
		(list (quote get_column)
			(if (qpu-low-scalar-helper-alias? tv) "__scalar_domain" tv)
			ti col ci)
		(cons head args)
		(cons
			(qpu-low-scalar-domain-normalize-expr head)
			(map (coalesceNil args '()) qpu-low-scalar-domain-normalize-expr))
		(if (qpu-low-scalar-helper-alias? expr) "__scalar_domain" expr))))

(define qpu-low-scalar-domain-normalize-table (lambda (td)
	(match td
		'(alias_ schema_ source_ is_outer_ join_expr_)
		(list
			(if (qpu-low-scalar-helper-alias? alias_) "__scalar_domain" alias_)
			schema_
			(if (qpp-tuple? source_)
				(qpu-low-scalar-domain-signature-tuple source_)
				(qpu-low-scalar-domain-normalize-expr source_))
			is_outer_
			(qpu-low-scalar-domain-normalize-expr join_expr_))
		td)))

(define qpu-low-scalar-domain-signature-tuple (lambda (tuple)
	(if (not (qpp-tuple? tuple))
		tuple
		(list
			(qpp-tuple-schema tuple)
			(map (coalesceNil (qpp-tuple-tables tuple) '())
				qpu-low-scalar-domain-normalize-table)
			(filter (qpp-fields-to-pairs (qpp-tuple-fields tuple))
				(lambda (pair) (match pair
					'(name _) (qpu-low-domain-key-field-name? name)
					false)))
			(qpu-low-scalar-domain-normalize-expr (qpp-tuple-condition tuple))
			(map (coalesceNil (qpp-tuple-group tuple) '())
				qpu-low-scalar-domain-normalize-expr)
			(qpu-low-scalar-domain-normalize-expr (qpp-tuple-having tuple))
			(map (coalesceNil (qpp-tuple-order tuple) '()) (lambda (item) (match item
				'(expr dir) (list (qpu-low-scalar-domain-normalize-expr expr) dir)
				item)))
			(qpp-tuple-limit tuple)
			(qpp-tuple-offset tuple)))))

(define qpu-low-field-name-for-expr (lambda (fields expr)
	(reduce (coalesceNil fields '()) (lambda (found pair) (match pair
		'(name fexpr)
		(if (and (nil? found) (equal? fexpr expr)) name found)
		found))
		nil)))

(define qpu-low-first-payload-field-name (lambda (fields)
	(reduce (coalesceNil fields '()) (lambda (found pair) (match pair
		'(name _)
		(if (not (nil? found))
			found
			(if (and
				(string? name)
				(>= (strlen name) 10)
				(equal? (substr name 0 10) "__payload_"))
				name
				nil))
		found))
		nil)))

(define qpu-low-payload-field-name (lambda (rhs-alias field-name expr existing-fields)
	(qpu-low-unique-projection-name
		(concat "__payload_" (fnv_hash (string
			(list rhs-alias field-name
				(qpu-low-scalar-domain-normalize-expr expr)))))
		existing-fields)))

(define qpu-low-helper-field-map-key (lambda (alias_ field-name)
	(concat (string alias_) "\0" (string field-name))))

(define qpu-low-rewrite-helper-alias-field-expr (lambda (expr alias-map)
	(match expr
		'(get_column tv ti col ci)
		(match (get_assoc alias-map (qpu-low-helper-field-map-key tv col))
			'(new-tv new-col) (list (quote get_column) new-tv ti new-col ci)
			expr)
		'((symbol get_column) tv ti col ci)
		(match (get_assoc alias-map (qpu-low-helper-field-map-key tv col))
			'(new-tv new-col) (list (quote get_column) new-tv ti new-col ci)
			expr)
		'((quote get_column) tv ti col ci)
		(match (get_assoc alias-map (qpu-low-helper-field-map-key tv col))
			'(new-tv new-col) (list (quote get_column) new-tv ti new-col ci)
			expr)
		(cons head args)
		(cons
			(qpu-low-rewrite-helper-alias-field-expr head alias-map)
			(map (coalesceNil args '()) (lambda (arg)
				(qpu-low-rewrite-helper-alias-field-expr arg alias-map))))
		expr)))

(define qpu-low-rewrite-helper-alias-field-fields (lambda (fields alias-map)
	(map (coalesceNil fields '()) (lambda (pair) (match pair
		'(name expr)
		(list name (qpu-low-rewrite-helper-alias-field-expr expr alias-map))
		pair)))))

(define qpu-low-scalar-value-ref? (lambda (expr alias_)
	(match expr
		'(get_column tv _ col _) (and (equal? tv alias_) (equal?? col "value"))
		'((symbol get_column) tv _ col _) (and (equal? tv alias_) (equal?? col "value"))
		'((quote get_column) tv _ col _) (and (equal? tv alias_) (equal?? col "value"))
		false)))

(define qpu-low-rewrite-duplicate-scalar-value-fields (lambda (fields alias_ payload-name)
	(if (nil? payload-name)
		fields
		(nth (reduce (coalesceNil fields '()) (lambda (state pair) (match pair
			'(name expr)
			(if (qpu-low-scalar-value-ref? expr alias_)
				(if (nth state 0)
					(list true
						(merge (nth state 1) (list
							(list name (list (quote get_column) alias_ false payload-name false)))))
					(list true (merge (nth state 1) (list pair))))
				(list (nth state 0) (merge (nth state 1) (list pair))))
			(list (nth state 0) (merge (nth state 1) (list pair)))))
			(list false '()))
			1))))

(define qpu-low-rewrite-alias-map-expr (lambda (expr alias-map)
	(match expr
		'(get_column tv ti col ci)
		(list (quote get_column)
			(coalesceNil (get_assoc alias-map (string tv)) tv)
			ti col ci)
		'((symbol get_column) tv ti col ci)
		(list (quote get_column)
			(coalesceNil (get_assoc alias-map (string tv)) tv)
			ti col ci)
		'((quote get_column) tv ti col ci)
		(list (quote get_column)
			(coalesceNil (get_assoc alias-map (string tv)) tv)
			ti col ci)
		(cons head args)
		(cons
			(qpu-low-rewrite-alias-map-expr head alias-map)
			(map (coalesceNil args '()) (lambda (arg)
				(qpu-low-rewrite-alias-map-expr arg alias-map))))
		expr)))

(define qpu-low-rewrite-alias-map-fields (lambda (fields alias-map)
	(map (coalesceNil fields '()) (lambda (pair) (match pair
		'(name expr)
		(list name (qpu-low-rewrite-alias-map-expr expr alias-map))
		pair)))))

(define qpu-low-scalar-domain-table-signature (lambda (td)
	(qpu-low-scalar-domain-normalize-table td)))

(define qpu-low-scalar-domain-alias-map (lambda (old-tuple new-tuple)
	(reduce (coalesceNil (qpp-tuple-tables new-tuple) '()) (lambda (acc new-td) (match new-td
		'(new-alias _ _ _ _)
		(if (not (qpu-low-scalar-helper-alias? new-alias))
			acc
			(begin
				(define new-sig (qpu-low-scalar-domain-table-signature new-td))
				(define old-alias
					(reduce (coalesceNil (qpp-tuple-tables old-tuple) '()) (lambda (found old-td)
						(if (not (nil? found))
							found
							(match old-td
								'(old-alias _ _ _ _)
								(if (and
									(qpu-low-scalar-helper-alias? old-alias)
									(equal? new-sig
										(qpu-low-scalar-domain-table-signature old-td)))
									old-alias
									nil)
								nil)))
						nil))
				(if (nil? old-alias)
					acc
					(set_assoc acc (string new-alias) old-alias))))
		acc))
		'())))

(define qpu-low-merge-scalar-domain-fields (lambda (existing-fields new-fields new-alias existing-alias) (begin
	(define initial-state (list (coalesceNil existing-fields '()) '()))
	(reduce (coalesceNil new-fields '()) (lambda (state pair) (match pair
		'(name expr)
		(if (qpu-low-domain-key-field-name? name)
			state
			(begin
				(define fields (nth state 0))
				(define rewrites (nth state 1))
				(define existing-name (qpu-low-field-name-for-expr fields expr))
				(if (not (nil? existing-name))
					(list fields
						(set_assoc rewrites
							(qpu-low-helper-field-map-key new-alias name)
							(list existing-alias existing-name)))
					(begin
						(define payload-name
							(qpu-low-payload-field-name new-alias name expr fields))
						(list
							(merge fields (list (list payload-name expr)))
							(set_assoc rewrites
								(qpu-low-helper-field-map-key new-alias name)
								(list existing-alias payload-name)))))))
		state))
		initial-state))))

(define qpu-low-merge-equivalent-scalar-tables-in-source (lambda (old-source new-source) (begin
	(define initial-state (list old-source '()))
	(reduce (coalesceNil (qpp-tuple-tables new-source) '()) (lambda (state new-td) (match new-td
		'(new-alias _ new-table-source _ _)
		(if (not (qpu-low-scalar-helper-alias? new-alias))
			state
			(begin
				(define current-old-source (nth state 0))
				(define current-rewrites (nth state 1))
				(define new-sig (qpu-low-scalar-domain-table-signature new-td))
				(define old-match
					(reduce (coalesceNil (qpp-tuple-tables current-old-source) '()) (lambda (found old-td)
						(if (not (nil? found))
							found
							(match old-td
								'(old-alias _ old-table-source _ _)
								(if (and
									(qpu-low-scalar-helper-alias? old-alias)
									(equal? new-sig
										(qpu-low-scalar-domain-table-signature old-td)))
									old-td
									nil)
								nil)))
						nil))
				(if (nil? old-match)
					state
					(match old-match
						'(old-alias old-schema old-table-source old-outer old-join)
						(if (or
							(not (qpp-tuple? old-table-source))
							(not (qpp-tuple? new-table-source)))
							state
							(begin
								(define nested
									(qpu-low-merge-equivalent-scalar-tables-in-source
										old-table-source new-table-source))
								(define old-table-source-with-nested (nth nested 0))
								(define nested-rewrites (nth nested 1))
								(define merged-fields
									(qpu-low-merge-scalar-domain-fields
										(qpp-fields-to-pairs
											(qpp-tuple-fields old-table-source-with-nested))
										(qpu-low-rewrite-alias-map-fields
											(qpu-low-rewrite-helper-alias-field-fields
												(qpp-fields-to-pairs (qpp-tuple-fields new-table-source))
												nested-rewrites)
											(qpu-low-scalar-domain-alias-map
												old-table-source-with-nested new-table-source))
										new-alias old-alias))
								(define merged-table-source
									(qpp-rebuild-tuple
										(qpp-tuple-schema old-table-source-with-nested)
										(qpp-tuple-tables old-table-source-with-nested)
										(nth merged-fields 0)
										(qpp-tuple-condition old-table-source-with-nested)
										(qpp-tuple-group old-table-source-with-nested)
										(qpp-tuple-having old-table-source-with-nested)
										(qpp-tuple-order old-table-source-with-nested)
										(qpp-tuple-limit old-table-source-with-nested)
										(qpp-tuple-offset old-table-source-with-nested)))
								(define old-source-with-merged-table
									(qpp-rebuild-tuple
										(qpp-tuple-schema current-old-source)
										(qpu-low-replace-table-source
											(qpp-tuple-tables current-old-source)
											old-alias
											merged-table-source)
										(qpp-tuple-fields current-old-source)
										(qpp-tuple-condition current-old-source)
										(qpp-tuple-group current-old-source)
										(qpp-tuple-having current-old-source)
										(qpp-tuple-order current-old-source)
										(qpp-tuple-limit current-old-source)
										(qpp-tuple-offset current-old-source)))
								(list old-source-with-merged-table
									(merge current-rewrites nested-rewrites (nth merged-fields 1)))))
						state))))
		state))
		initial-state))))

(define qpu-low-merge-equivalent-scalar-derived (lambda (left-tuple derived-entry rhs-alias)
	(match derived-entry
		'(new-alias new-schema new-source new-outer new-join)
		(if (not (qpp-tuple? new-source))
			nil
			(begin
				(define new-signature
					(qpu-low-scalar-domain-signature-tuple new-source))
				(reduce (coalesceNil (qpp-tuple-tables left-tuple) '()) (lambda (found td)
					(if (not (nil? found))
						found
						(match td
							'(old-alias old-schema old-source old-outer old-join)
							(if (and
								(qpu-low-scalar-helper-alias? old-alias)
								(qpp-tuple? old-source)
								(equal? new-signature
									(qpu-low-scalar-domain-signature-tuple old-source)))
								(begin
									(define nested-merge
										(qpu-low-merge-equivalent-scalar-tables-in-source
											old-source new-source))
									(define old-source-with-nested (nth nested-merge 0))
									(define nested-rewrites (nth nested-merge 1))
									(define merged
										(qpu-low-merge-scalar-domain-fields
											(qpp-fields-to-pairs (qpp-tuple-fields old-source-with-nested))
											(qpu-low-rewrite-alias-map-fields
												(qpu-low-rewrite-helper-alias-field-fields
													(qpp-fields-to-pairs (qpp-tuple-fields new-source))
													nested-rewrites)
												(qpu-low-scalar-domain-alias-map
													old-source-with-nested new-source))
											new-alias old-alias))
									(define merged-source
										(qpp-rebuild-tuple
											(qpp-tuple-schema old-source-with-nested)
											(qpp-tuple-tables old-source-with-nested)
											(nth merged 0)
											(qpp-tuple-condition old-source-with-nested)
											(qpp-tuple-group old-source-with-nested)
											(qpp-tuple-having old-source-with-nested)
											(qpp-tuple-order old-source-with-nested)
											(qpp-tuple-limit old-source-with-nested)
											(qpp-tuple-offset old-source-with-nested)))
									(define merged-rewrites
										(merge nested-rewrites (nth merged 1)))
									(list old-alias merged-source
										merged-rewrites))
								nil)
							nil)))
					nil)))
		nil)))

(define qpu-low-replace-table-source (lambda (tables target new-source)
	(map (coalesceNil tables '()) (lambda (td) (match td
		'(alias_ schema_ source_ is_outer_ join_expr_)
		(if (equal? alias_ target)
			(list alias_ schema_ new-source is_outer_ join_expr_)
			td)
		td)))))

/* qpu-low-sq-rewrites — session map of pending scalar-helper field → target-expr
substitutions populated by qpu-low-join-inline-scalar. Consumed by
qpu-low-select / qpu-low-map / qpu-low-groupby when they wrap the inline-
flat result; the outer's WHERE/projections/keys may contain refs to
helper.value (from lift's marker substitution) that the inline-flat path
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
the inlined scalar-helper table is actually visible, preventing a nested scalar's
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

(define qpu-low-local-value-guard? (lambda (pred alias_)
	(and
		(expr_uses_session_state pred)
		(not (qpu-low-expr-has-scalar-helper-ref? pred))
		(reduce (extract_tblvars pred) (lambda (ok tv)
			(and ok (equal? tv alias_))) true))))

(define qpu-low-push-local-value-guard-field (lambda (state pair alias_)
	(match pair
		'(name expr)
		(if (qpu-low-domain-key-field? pair)
			(list (nth state 0) (merge (nth state 1) (list pair)))
			(match expr
				'((quote if) pred value nil)
				(if (qpu-low-local-value-guard? pred alias_)
					(list
						(qpu-low-and-cond (nth state 0) pred)
						(merge (nth state 1) (list (list name value))))
					(list (nth state 0) (merge (nth state 1) (list pair))))
				'((symbol if) pred value nil)
				(if (qpu-low-local-value-guard? pred alias_)
					(list
						(qpu-low-and-cond (nth state 0) pred)
						(merge (nth state 1) (list (list name value))))
					(list (nth state 0) (merge (nth state 1) (list pair))))
				(list (nth state 0) (merge (nth state 1) (list pair)))))
		(list (nth state 0) (merge (nth state 1) (list pair))))))

(define qpu-low-rhs-local-guard? (lambda (pred rhs-alias)
	(and
		(expr_uses_session_state pred)
		(reduce (extract_tblvars pred) (lambda (ok tv)
			(and ok (equal? tv rhs-alias))) true))))

(define qpu-low-pull-rhs-local-value-guards-expr (lambda (expr rhs-alias)
	(match expr
		'((quote if) pred value nil)
		(if (qpu-low-rhs-local-guard? pred rhs-alias)
			(begin
				(define value-result
					(qpu-low-pull-rhs-local-value-guards-expr value rhs-alias))
				(list (qpu-low-and-cond pred (nth value-result 0)) (nth value-result 1)))
			(list true expr))
		'((symbol if) pred value nil)
		(if (qpu-low-rhs-local-guard? pred rhs-alias)
			(begin
				(define value-result
					(qpu-low-pull-rhs-local-value-guards-expr value rhs-alias))
				(list (qpu-low-and-cond pred (nth value-result 0)) (nth value-result 1)))
			(list true expr))
		(cons head args)
		(begin
			(define pulled-args
				(reduce (coalesceNil args '()) (lambda (state arg)
					(begin
						(define arg-result
							(qpu-low-pull-rhs-local-value-guards-expr arg rhs-alias))
						(list
							(qpu-low-and-cond (nth state 0) (nth arg-result 0))
							(merge (nth state 1) (list (nth arg-result 1))))))
					(list true '())))
			(list (nth pulled-args 0) (cons head (nth pulled-args 1))))
		(list true expr))))

(define qpu-low-pull-rhs-local-value-guards-fields (lambda (fields rhs-alias)
	(reduce (coalesceNil fields '()) (lambda (state pair)
		(match pair
			'(name expr)
			(begin
				(define expr-result
					(qpu-low-pull-rhs-local-value-guards-expr expr rhs-alias))
				(list
					(qpu-low-and-cond (nth state 0) (nth expr-result 0))
					(merge (nth state 1) (list (list name (nth expr-result 1))))))
			(list (nth state 0) (merge (nth state 1) (list pair)))))
		(list true '()))))

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
						(and
							(not (equal? (count ord) 0))
							(not (qpu-low-order-only-alias? ord key-alias)))
						(qpp-tuple? td-tname)
						(qpu-low-scan-tagged-table? td-tname))
						sub-tuple
						(begin
							(define tuple-cond (qpp-tuple-condition sub-tuple))
							(define guard-result
								(reduce flds (lambda (state pair)
									(qpu-low-push-local-value-guard-field
										state pair td-alias))
									(list tuple-cond '())))
							(define pushed-tuple-cond (nth guard-result 0))
							(define pushed-fields (nth guard-result 1))
							(define tuple-cond-trivial
								(or (nil? pushed-tuple-cond) (equal? pushed-tuple-cond true)
									(equal? pushed-tuple-cond (quote true))))
							(define tuple-cond-first-table-only
								(or tuple-cond-trivial
									(and
										(expr_uses_session_state pushed-tuple-cond)
										(reduce (extract_tblvars pushed-tuple-cond) (lambda (ok tv)
											(and ok (equal? tv td-alias))) true))))
							(if (not tuple-cond-first-table-only)
								sub-tuple
								(begin
									(define exists-limit-one?
										(qpu-low-expr-has-exists-value-payload?
											pushed-fields))
									(define partition-order
										(map key-projs (lambda (kp)
											(list (nth kp 2) (quote <)))))
									(define no-order-limit?
										(equal? (count ord) 0))
									(define tagged (make_scan_tagged_table
										td-tname
										(merge partition-order ord)
										(if (and no-order-limit?
											(not exists-limit-one?))
											nil
											1)
										nil
										(count partition-order)
										1))
									(define tagged-je
										(if tuple-cond-trivial
											td-je
											(qpu-low-and-cond td-je pushed-tuple-cond)))
									(qpp-rebuild-tuple
										(qpp-tuple-schema sub-tuple)
										(cons (list td-alias td-schema tagged td-io tagged-je)
											(cdr tbls))
										pushed-fields
										(if tuple-cond-trivial pushed-tuple-cond true)
										(qpp-tuple-group sub-tuple)
										(qpp-tuple-having sub-tuple)
										(qpp-tuple-order sub-tuple)
										(qpp-tuple-limit sub-tuple)
										(qpp-tuple-offset sub-tuple)))))
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
		`(t4 id)` to `(domain_scalar_* id)` produces a reference to a non-existent
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
		/* For UNCORRELATED scalar-helper derived: tag the inner base
		table with scan-tagged-table once_limit=2 so multi-row inner scans
		error per FAQ §20. CORRELATED scalars: tried partition-tagging in
		tick 12 of session 2026-05-31 — regressed -30 tests. The flat
		derived path doesn't have enough scope to set partition_cols
		correctly without breaking sibling queries. Correlated remains on
		the existing derived-wrap path until a proper inline-flat refactor
		restructures the IR (FAQ §43 plan). */
		(define is-scalar-rhs
			(qpu-low-scalar-helper-alias? rhs-alias))
		(define is-uncorrelated
			(or (nil? join-pred) (equal? join-pred true)
				(equal? join-pred (quote true))))
		(define right-tuple-keys
			(if (and is-scalar-rhs is-uncorrelated)
				(qpu-low-tag-inner-once-limit right-tuple-keys-dedup)
				(if (and is-scalar-rhs is-left (not is-uncorrelated))
					(if (nil? (qpp-tuple-limit right-tuple-keys-dedup))
						right-tuple-keys-dedup
						(qpu-low-tag-correlated-ordered-scalar-limit
							right-tuple-keys-dedup key-rename-map))
					right-tuple-keys-dedup)))
		(define right-condition-can-hoist
			(and is-scalar-rhs is-left
				(equal? (count (coalesceNil (qpp-tuple-group right-tuple-keys) '())) 0)
				(nil? (qpp-tuple-having right-tuple-keys))))
		(define right-tuple-join-safe
			(if right-condition-can-hoist
				(qpu-low-push-single-alias-condition-to-tables right-tuple-keys)
				right-tuple-keys))
		(define right-tuple-hoisted-cond
			(if right-condition-can-hoist
				(qpp-tuple-condition right-tuple-join-safe)
				true))
		(define right-tuple-final
			(if right-condition-can-hoist
				(qpp-rebuild-tuple
					(qpp-tuple-schema right-tuple-join-safe)
					(qpp-tuple-tables right-tuple-join-safe)
					(qpp-tuple-fields right-tuple-join-safe)
					true
					(qpp-tuple-group right-tuple-join-safe)
					(qpp-tuple-having right-tuple-join-safe)
					(qpp-tuple-order right-tuple-join-safe)
					(qpp-tuple-limit right-tuple-join-safe)
					(qpp-tuple-offset right-tuple-join-safe))
				right-tuple-join-safe))
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
		(define pulled-field-guards
			(if (and is-left is-scalar-rhs)
				(qpu-low-pull-rhs-local-value-guards-fields
					rewritten-fields rhs-alias)
				(list true rewritten-fields)))
		(define rewritten-fields-final (nth pulled-field-guards 1))
		(define pulled-field-guard-cond (nth pulled-field-guards 0))
		(define rewritten-cond (qpu-low-rewrite-refs
			(qpp-tuple-condition left-tuple) right-only-aliases rhs-alias))
		(define raw-rewritten-having
			(if (nil? (qpp-tuple-having left-tuple))
				nil
				(qpu-low-rewrite-refs
					(qpp-tuple-having left-tuple) right-only-aliases rhs-alias)))
		(define rewritten-having
			(if (nil? raw-rewritten-having)
				nil
				(qpu-low-rewrite-cols-by-renames
					raw-rewritten-having key-rename-map)))
		/* Rewrite join-pred using the rename-map so synthesized __kt_* names
		are referenced under rhs-alias. Refs not in rename-map use the
		regular rewrite (alias bump). */
		(define rewritten-pred (qpu-low-rewrite-refs
			local-rewritten-pred
			right-source-aliases rhs-alias))
		(define normalized-rewritten-pred-raw
			(qpu-simplify-predicate
				(qpu-low-normalize-kt-refs-in-expr
					(qpu-low-and-cond rewritten-pred
						(qpu-low-rewrite-by-renames
							right-tuple-hoisted-cond key-rename-map rhs-alias))
					(list rhs-alias)
					(qpp-fields-to-pairs (qpp-tuple-fields right-tuple-final)))))
		(define nested-key-hoist
			(list right-tuple-final normalized-rewritten-pred-raw))
		(define right-tuple-final2 (nth nested-key-hoist 0))
		(define normalized-rewritten-pred-base (nth nested-key-hoist 1))
		(define normalized-rewritten-pred
			(if (and is-left is-scalar-rhs)
				(qpu-and-from-conjuncts
					(map (qpu-and-conjuncts normalized-rewritten-pred-base)
						(lambda (conj)
							(qpu-low-null-tolerate-nested-scalar-key-conjunct
								conj rhs-alias
								(qpp-tuple-fields right-tuple-final2)))))
				normalized-rewritten-pred-base))
		(define scalar-limit-one-nested-projection?
			(and
				(equal? (qpp-tuple-limit right-tuple-final2) 1)
				(nil? (qpp-tuple-offset right-tuple-final2))
				(equal? (coalesceNil (qpp-tuple-order right-tuple-final2) '()) '())
				(qpu-low-expr-has-scalar-helper-ref?
					(qpp-tuple-fields right-tuple-final2))
				(qpu-low-expr-refs-alias?
					(qpp-tuple-fields left-tuple) rhs-alias)))
		(define scalar-limit-one-exists-domain?
			(and
				(equal? (qpp-tuple-limit right-tuple-final2) 1)
				(nil? (qpp-tuple-offset right-tuple-final2))
				(equal? (coalesceNil (qpp-tuple-order right-tuple-final2) '()) '())
				(qpu-low-expr-has-exists-value-payload?
					(qpp-tuple-fields right-tuple-final2))))
		(define scalar-grouped-domain?
			(and
				(not is-uncorrelated)
				(qpu-low-expr-refs-alias?
					(qpp-tuple-fields left-tuple) rhs-alias)
				(or
					(> (count (coalesceNil (qpp-tuple-group right-tuple-final2) '())) 0)
					(not (nil? (qpp-tuple-having right-tuple-final2)))
					(qpu-low-fields-have-aggregate?
						(qpp-tuple-fields right-tuple-final2)))))
		(define derived-local-aliases (merge left-source-aliases (list rhs-alias)))
		(define scoped-normalized-rewritten-pred
			(if is-left
				(qpu-low-add-outer-hop-for-nonlocal-refs
					normalized-rewritten-pred derived-local-aliases)
				normalized-rewritten-pred))
		(define scoped-normalized-rewritten-pred-final
			(qpu-low-and-cond
				scoped-normalized-rewritten-pred
				pulled-field-guard-cond))
		(define right-tuple-derived
			(if (and is-left is-scalar-rhs
				(or
					scalar-limit-one-nested-projection?
					scalar-limit-one-exists-domain?
					scalar-grouped-domain?
					(qpu-low-pred-uses-nested-scalar-key?
						normalized-rewritten-pred rhs-alias
						(qpp-tuple-fields right-tuple-final2))
					(qpu-low-tuple-join-refs-aliases?
						right-tuple-final2 left-source-aliases)))
				(qpu-low-domainize-nested-scalar-derived
					left-tuple right-tuple-final2 rhs-alias
					normalized-rewritten-pred)
				right-tuple-final2))
		(define pulled-derived-field-guards
			(if (and is-left is-scalar-rhs)
				(qpu-low-pull-rhs-local-value-guards-fields
					(qpp-tuple-fields right-tuple-derived) rhs-alias)
				(list true (qpp-tuple-fields right-tuple-derived))))
		(define right-tuple-derived-final
			(qpp-rebuild-tuple
				(qpp-tuple-schema right-tuple-derived)
				(qpp-tuple-tables right-tuple-derived)
				(nth pulled-derived-field-guards 1)
				(qpp-tuple-condition right-tuple-derived)
				(qpp-tuple-group right-tuple-derived)
				(qpp-tuple-having right-tuple-derived)
				(qpp-tuple-order right-tuple-derived)
				(qpp-tuple-limit right-tuple-derived)
				(qpp-tuple-offset right-tuple-derived)))
		(define scoped-normalized-rewritten-pred-final2
			(qpu-low-and-cond
				scoped-normalized-rewritten-pred-final
				(nth pulled-derived-field-guards 0)))
		(define derived-join-aliases
			(merge left-source-aliases (list rhs-alias)))
		(define scoped-normalized-rewritten-pred-final3
			(qpu-and-from-conjuncts
				(filter
					(qpu-and-conjuncts
						(qpu-low-filter-cond-by-aliases
							scoped-normalized-rewritten-pred-final2
							derived-join-aliases))
					(lambda (conj)
						(and
							(not (qpu-low-expr-has-explicit-outer? conj))
							(not (qpu-low-expr-has-ref-outside?
								conj derived-join-aliases))
							(not (and is-left is-scalar-rhs
								(has? (extract_tblvars conj) rhs-alias)
								(not (qpu-low-equality-conjunct? conj))))
							(not (and is-left is-scalar-rhs
								(qpu-low-has-conjunct? rewritten-cond conj))))))))
		(define derived-entry
			(if is-left
				/* LEFT join: derived table is isOuter=true with joinExpr = pred.
				Per-key misses get NULL-extended automatically by the scan
				infrastructure (FAQ §22 isOuter contract). */
				(list rhs-alias (qpp-tuple-schema right-tuple-derived-final)
					right-tuple-derived-final true scoped-normalized-rewritten-pred-final3)
				/* INNER join: derived table is plain; predicate flows into WHERE. */
				(list rhs-alias (qpp-tuple-schema right-tuple-derived-final)
					right-tuple-derived-final false nil)))
		(define final-cond
			(if is-left
				/* LEFT: predicate is in joinExpr, NOT in WHERE (else inner-join semantics). */
				rewritten-cond
				(qpu-low-and-cond rewritten-cond scoped-normalized-rewritten-pred-final2)))
		(define scalar-domain-merge
			(if (and is-left is-scalar-rhs)
				(qpu-low-merge-equivalent-scalar-derived
					left-tuple derived-entry rhs-alias)
				nil))
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
						(rewrite-post-group-expr rewritten-having)
						(if is-left true normalized-rewritten-pred))
					nil
					nil
					(qpp-tuple-order left-tuple)
					(qpp-tuple-limit left-tuple)
					(qpp-tuple-offset left-tuple)))
			(qpp-rebuild-tuple
				(qpp-tuple-schema left-tuple)
				(if (nil? scalar-domain-merge)
					(merge (qpp-tuple-tables left-tuple) (list derived-entry))
					(qpu-low-replace-table-source
						(qpp-tuple-tables left-tuple)
						(nth scalar-domain-merge 0)
						(nth scalar-domain-merge 1)))
				(if (nil? scalar-domain-merge)
					rewritten-fields-final
					(qpu-low-rewrite-helper-alias-field-fields
						rewritten-fields-final
						(nth scalar-domain-merge 2)))
				(if (nil? scalar-domain-merge)
					final-cond
					(qpu-low-rewrite-helper-alias-field-expr
						final-cond
						(nth scalar-domain-merge 2)))
				(qpp-tuple-group left-tuple)
				(if (or (nil? rewritten-having) (nil? scalar-domain-merge))
					rewritten-having
					(qpu-low-rewrite-helper-alias-field-expr
						rewritten-having
						(nth scalar-domain-merge 2)))
				(if (nil? scalar-domain-merge)
					(qpp-tuple-order left-tuple)
					(map (coalesceNil (qpp-tuple-order left-tuple) '()) (lambda (order-item) (match order-item
						'(order-expr order-dir)
						(list
							(qpu-low-rewrite-helper-alias-field-expr
								order-expr
								(nth scalar-domain-merge 2))
							order-dir)
						order-item))))
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
