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

/* qpu-low-find-deep-alias-in-tables — for an alias name NOT in the top-level
table aliases, find which top-level DERIVED table's sub-tuple contains it.
Returns the derived-alias name, or nil if alias isn't reachable through any
of the top-level deriveds. Used by qpu-low-ensure-join-key-fields to cascade
__kt projections through nested rhs-aliased wrappers (FAQ §42 step 2/2). */
(define qpu-low-find-deep-alias-in-tables (lambda (tables alias)
	(reduce (coalesceNil tables '()) (lambda (acc td) (match td
		'(td-alias td-schema td-tname td-isOuter td-jE)
			(if (not (nil? acc)) acc
				(if (qpp-tuple? td-tname)
					(begin
						(define inner-aliases (map
							(coalesceNil (qpp-tuple-tables td-tname) '())
							(lambda (itd) (if (or (nil? itd) (< (count itd) 1))
								nil (nth itd 0)))))
						(if (has? inner-aliases alias) td-alias nil))
					nil))
		acc)) nil)))

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
					(define existing-name (qpu-low-fields-find-by-expr sub-fields tv col))
					(if (not (nil? existing-name))
						(begin (result-name "n" existing-name) td)
						(begin
							(define synthesized (concat "__kt_" col))
							(define unique-name (qpu-low-unique-projection-name
								synthesized sub-fields))
							(result-name "n" unique-name)
							(define new-sub-fields (merge sub-fields
								(list (list unique-name
									(list (quote get_column) tv false col false)))))
							(define new-sub (qpp-rebuild-tuple
								(qpp-tuple-schema sub)
								(qpp-tuple-tables sub)
								new-sub-fields
								(qpp-tuple-condition sub)
								(qpp-tuple-group sub)
								(qpp-tuple-having sub)
								(qpp-tuple-order sub)
								(qpp-tuple-limit sub)
								(qpp-tuple-offset sub)))
							(list td-alias td-schema new-sub td-isOuter td-jE))))
				td)
		td))))
	(list updated-tables (result-name "n")))))

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
						(define existing-name (qpu-low-fields-find-by-expr
							(merge existing-fields (nth acc 1)) tv col))
						(if (not (nil? existing-name))
							(list (nth acc 0) (nth acc 1) (merge (nth acc 2)
								(list (list (list tv col) existing-name))))
							(begin
								(define synthesized (concat "__kt_" col))
								(define unique-name (qpu-low-unique-projection-name
									synthesized (merge existing-fields (nth acc 1))))
								(list
									(nth acc 0)
									(merge (nth acc 1)
										(list (list unique-name
											(list (quote get_column) tv false col false))))
									(merge (nth acc 2)
										(list (list (list tv col) unique-name)))))))
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
		(list
			(if (and (equal? (count added-projections) 0)
					 (equal? final-tables top-tables))
				right-tuple
				(qpp-rebuild-tuple
					(qpp-tuple-schema right-tuple)
					final-tables
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
				(qpu-low-rewrite-map-projections (qpir-map-projections node) wrap-alias)
				true
				nil nil nil nil nil))
		/* No GROUP BY: standard replace-fields path. */
		(qpp-rebuild-tuple
			(qpp-tuple-schema child-tuple)
			(qpp-tuple-tables child-tuple)
			(qpir-map-projections node)
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
			(if (and (not outer-has-group) (not outer-has-agg-fields)
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
		(if (or has-lim has-agg (not (equal? (count tbls) 1)))
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
		(qpp-rebuild-tuple
			(qpp-tuple-schema left-tuple)
			(merge (qpp-tuple-tables left-tuple) (list new-entry))
			rewritten-fields
			rewritten-cond
			(qpp-tuple-group left-tuple)
			(qpp-tuple-having left-tuple)
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

(define qpu-low-tag-has-count-distinct? (lambda (expr) (match expr
	(cons head args) (or
		(equal? head (quote count_distinct))
		(equal? head (symbol count_distinct))
		(qpu-low-tag-has-count-distinct? head)
		(reduce (coalesceNil args '()) (lambda (acc a)
			(or acc (qpu-low-tag-has-count-distinct? a))) false))
	false)))

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
				(qpu-low-tag-inner-once-limit right-tuple-keys-raw)
				right-tuple-keys-raw))
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
	(qpu-low-sq-rewrites-clear)
	(define lowered (qpu-lower-to-tuple qpir-tree))
	(if (not (qpp-tuple? lowered))
		(error "lower_to_scans_pass: lowering did not produce a 7-tuple")
		lowered))))
