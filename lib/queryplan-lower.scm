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
			(list (quote get_column) to-alias false col false)
			expr)
		'((quote get_column) tv ti col ci)
		(if (has? from-aliases tv)
			(list (quote get_column) to-alias false col false)
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

/* qpu-low-ensure-join-key-fields — for every `(tv col)` referenced by
`join-pred` against the right-tuple's underlying tables, ensure the
right-tuple's fields list projects it under SOME name. Returns a pair
(updated-right-tuple, rename-map) where rename-map is an assoc list of
((tv col) projected-name) so the caller can rewrite join-pred refs to use
the canonical projected name (which may differ from `col` if `col` was
already taken by a different expression).

Without this, the join predicate retargeting `(t4 id)` → `(sq_N id)` produces
a reference to a non-existent field of the derived sub-tuple — the legacy
lower-level engine then returns nil for that lookup and the join never
matches (per FAQ §22 LEFT-JOIN semantics, every outer row gets NULL-extended).

Naming: we use a synthesized name `__kt_<col>` to avoid colliding with any
existing projection name in the sub. The rename-map ensures join-pred refs
get retargeted to this name. */
(define qpu-low-ensure-join-key-fields (lambda (right-tuple join-pred right-source-aliases)
	(begin
		(define needed-refs (qpu-low-collect-refs-for-aliases join-pred right-source-aliases))
		(define existing-fields (qpp-fields-to-pairs (qpp-tuple-fields right-tuple)))
		(define plan (reduce needed-refs (lambda (acc ref) (match ref
			'(tv col)
				(begin
					(define existing-name (qpu-low-fields-find-by-expr existing-fields tv col))
					(if (not (nil? existing-name))
						/* Already projected under existing-name — reuse it. */
						(list (nth acc 0) (merge (nth acc 1)
							(list (list (list tv col) existing-name))))
						(begin
							(define synthesized (concat "__kt_" col))
							/* If synthesized name already taken or already added to acc, suffix it. */
							(define unique-name (qpu-low-unique-projection-name
								synthesized (merge existing-fields (nth acc 0))))
							(list (merge (nth acc 0)
									(list (list unique-name
										(list (quote get_column) tv false col false))))
								(merge (nth acc 1)
									(list (list (list tv col) unique-name)))))))
			acc)) (list (list) (list))))
		(define added-projections (nth plan 0))
		(define rename-map (nth plan 1))
		(list
			(if (equal? (count added-projections) 0) right-tuple
				(qpp-rebuild-tuple
					(qpp-tuple-schema right-tuple)
					(qpp-tuple-tables right-tuple)
					(merge existing-fields added-projections)
					(qpp-tuple-condition right-tuple)
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
	(define new-cond (qpu-low-and-cond
		(qpp-tuple-condition child-tuple)
		(qpir-select-predicate node)))
	(qpp-rebuild-tuple
		(qpp-tuple-schema child-tuple)
		(qpp-tuple-tables child-tuple)
		(qpp-tuple-fields child-tuple)
		new-cond
		(qpp-tuple-group child-tuple)
		(qpp-tuple-having child-tuple)
		(qpp-tuple-order child-tuple)
		(qpp-tuple-limit child-tuple)
		(qpp-tuple-offset child-tuple)))))

(define qpu-low-map (lambda (node) (begin
	(define child-tuple (qpu-lower-to-tuple (qpir-map-child node)))
	(qpp-rebuild-tuple
		(qpp-tuple-schema child-tuple)
		(qpp-tuple-tables child-tuple)
		(qpir-map-projections node)
		(qpp-tuple-condition child-tuple)
		(qpp-tuple-group child-tuple)
		(qpp-tuple-having child-tuple)
		(qpp-tuple-order child-tuple)
		(qpp-tuple-limit child-tuple)
		(qpp-tuple-offset child-tuple)))))

(define qpu-low-groupby (lambda (node) (begin
	(define child-tuple (qpu-lower-to-tuple (qpir-groupby-child node)))
	/* Combine child's existing fields (the projected base columns) with the
	   group-by key projections and the aggregate projections. The resulting
	   7-tuple's fields list is what the GROUP BY query exposes. */
	(define key-projections (map (qpir-groupby-keys node) qpu-low-key-projection))
	(define agg-projections (qpir-groupby-aggs node))
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
	(if (nil? rhs-alias)
		(qpu-low-join-merge-tables left-tuple right-tuple join-pred)
		(qpu-low-join-wrap-derived left-tuple right-tuple join-pred rhs-alias
			(qpir-join-type node))))))

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
		(define right-tuple-keys (nth keys-result 0))
		(define key-rename-map (nth keys-result 1))
		(define rewritten-fields (qpu-low-rewrite-projections
			(qpp-tuple-fields left-tuple) right-only-aliases rhs-alias))
		(define rewritten-cond (qpu-low-rewrite-refs
			(qpp-tuple-condition left-tuple) right-only-aliases rhs-alias))
		/* Rewrite join-pred using the rename-map so synthesized __kt_* names
		   are referenced under rhs-alias. Refs not in rename-map use the
		   regular rewrite (alias bump). */
		(define rewritten-pred (qpu-low-rewrite-refs
			(qpu-low-rewrite-by-renames join-pred key-rename-map rhs-alias)
			right-source-aliases rhs-alias))
		(define is-left (equal? join-type (quote left)))
		(define derived-entry
			(if is-left
				/* LEFT join: derived table is isOuter=true with joinExpr = pred.
				   Per-key misses get NULL-extended automatically by the scan
				   infrastructure (FAQ §22 isOuter contract). */
				(list rhs-alias (qpp-tuple-schema right-tuple-keys)
					right-tuple-keys true rewritten-pred)
				/* INNER join: derived table is plain; predicate flows into WHERE. */
				(list rhs-alias (qpp-tuple-schema right-tuple-keys)
					right-tuple-keys false nil)))
		(define final-cond
			(if is-left
				/* LEFT: predicate is in joinExpr, NOT in WHERE (else inner-join semantics). */
				rewritten-cond
				(qpu-low-and-cond rewritten-cond rewritten-pred)))
		(qpp-rebuild-tuple
			(qpp-tuple-schema left-tuple)
			(merge (qpp-tuple-tables left-tuple) (list derived-entry))
			rewritten-fields
			final-cond
			(qpp-tuple-group left-tuple)
			(qpp-tuple-having left-tuple)
			(qpp-tuple-order left-tuple)
			(qpp-tuple-limit left-tuple)
			(qpp-tuple-offset left-tuple)))))

/* ==================== Public driver ==================== */

/* lower_to_scans_pass — the L4 transformation.
Takes a qpir tree (post-unnest, no dep-joins, F(root)=∅) and returns a
single 7-tuple compatible with the legacy build_queryplan_inner. The
caller then feeds this 7-tuple into the existing physical compiler for
scan/keytable/join emission. */
(define lower_to_scans_pass (lambda (qpir-tree) (begin
	(define lowered (qpu-lower-to-tuple qpir-tree))
	(if (not (qpp-tuple? lowered))
		(error "lower_to_scans_pass: lowering did not produce a 7-tuple")
		lowered))))
