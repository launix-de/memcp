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
		(define rewritten-fields (qpu-low-rewrite-projections
			(qpp-tuple-fields left-tuple) right-source-aliases rhs-alias))
		(define rewritten-cond (qpu-low-rewrite-refs
			(qpp-tuple-condition left-tuple) right-source-aliases rhs-alias))
		(define rewritten-pred (qpu-low-rewrite-refs
			join-pred right-source-aliases rhs-alias))
		(define is-left (equal? join-type (quote left)))
		(define derived-entry
			(if is-left
				/* LEFT join: derived table is isOuter=true with joinExpr = pred.
				   Per-key misses get NULL-extended automatically by the scan
				   infrastructure (FAQ §22 isOuter contract). */
				(list rhs-alias (qpp-tuple-schema right-tuple)
					right-tuple true rewritten-pred)
				/* INNER join: derived table is plain; predicate flows into WHERE. */
				(list rhs-alias (qpp-tuple-schema right-tuple)
					right-tuple false nil)))
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
