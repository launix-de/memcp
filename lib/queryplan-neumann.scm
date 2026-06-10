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
== Public entry point — neumann_compile_select ==

Runs the full BTW2025 compiler pipeline on a parser-emitted SELECT 7-tuple
and returns a clean 7-tuple compatible with build_queryplan_inner /
untangle_query / the legacy emission path.

Pipeline (L1 → L4):
1. alias_normalize_pass     — drop NUL-separator provenance, pick visible aliases
2. column_resolve_pass      — resolve ti/ci flags against schemas
3. lift_dep_joins_pass      — every inner_select* marker becomes a qpir-dep-join
4. unnest_pass              — BTW2025 §3 holistic decorrelation; no dep-joins remain
5. lower_to_scans_pass      — qpir tree → single 7-tuple via derived-table wrapping

Output invariants:
- No inner_select / inner_select_in / inner_select_exists markers
- F(root) = ∅ (every column ref bound by some provider)
- No qpir-dep-join in the operator tree (eliminated by unnest)
- Derived tables only where required: qpir-groupby outputs (per FAQ §32
materialization rule — group caches must materialize)

The caller (build_queryplan_term, or any direct invocation) can feed this
output to untangle_query / build_queryplan_inner and get a normal physical
plan back. The whole "subselect handling" code path inside untangle_query
becomes inactive because there are no markers left.

This module exposes the public API. On this branch `build_queryplan_term`
routes SELECT 7-tuples through this pipeline as the standard path.
*/

/* ==================== Public API ==================== */

/* qpn-flatten-tuple-recursive — convert all fields lists in a 7-tuple from
list-of-pairs to flat (name1 expr1 name2 expr2 …), INCLUDING any derived
sub-tuples nested in the tables list. The legacy build_queryplan_inner /
untangle_query consumers iterate fields via extract_assoc / reduce_assoc
which assume the flat form. */
(define qpn-domain-key-field? (lambda (pair) (match pair
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

(define qpn-limit-one-value-reducer
	(list (quote lambda)
		(list (quote acc) (quote val))
		(list (quote if)
			(list (quote nil?) (quote val))
			(quote acc)
			(quote val))))

(define qpn-direct-scalar-helper-ref? (lambda (expr)
	(match expr
		'(get_column tv _ _ _)
		(qpp-scalar-helper-alias? tv)
		'((symbol get_column) tv _ _ _)
		(qpp-scalar-helper-alias? tv)
		'((quote get_column) tv _ _ _)
		(qpp-scalar-helper-alias? tv)
		false)))

(define qpn-collapse-local-limit-one-domain (lambda (t)
	(if (not (qpp-tuple? t)) t
		(begin
			(define fields (qpp-fields-to-pairs (qpp-tuple-fields t)))
			(define key-fields (filter fields qpn-domain-key-field?))
			(define value-fields (filter fields (lambda (pair)
				(not (qpn-domain-key-field? pair)))))
			(define direct-scalar-values
				(reduce value-fields (lambda (ok pair) (match pair
					'(_ expr) (and ok (qpn-direct-scalar-helper-ref? expr))
					false))
					true))
			(if (and
				(equal? (qpp-tuple-limit t) 1)
				(nil? (qpp-tuple-offset t))
				(equal? (coalesceNil (qpp-tuple-order t) '()) '())
				(equal? (coalesceNil (qpp-tuple-group t) '()) '())
				(nil? (qpp-tuple-having t))
				(> (count key-fields) 0)
				(> (count value-fields) 0)
				direct-scalar-values)
				(qpp-rebuild-tuple
					(qpp-tuple-schema t)
					(qpp-tuple-tables t)
					(map fields (lambda (pair) (match pair
						'(name expr)
						(if (qpn-domain-key-field? pair)
							pair
							(list name (list (quote aggregate) expr
								qpn-limit-one-value-reducer nil)))
						pair)))
					(qpp-tuple-condition t)
					(map key-fields (lambda (pair) (match pair
						'(_ expr) expr
						nil)))
					nil
					'()
					nil
					nil)
				t)))))

(define qpn-flatten-tuple-recursive (lambda (t)
	(if (not (qpp-tuple? t)) t
		(begin
			(define collapsed (qpn-collapse-local-limit-one-domain t))
			(qpp-rebuild-tuple
				(qpp-tuple-schema collapsed)
				(map (coalesceNil (qpp-tuple-tables collapsed) (list)) (lambda (td)
					(if (or (nil? td) (< (count td) 3)) td
						(begin
							(define tname (nth td 2))
							(if (qpp-tuple? tname)
								/* derived table: recursively flatten its 7-tuple */
								(merge (list (nth td 0) (nth td 1)
									(qpn-flatten-tuple-recursive tname))
									(if (>= (count td) 4)
										(list (nth td 3))
										(list false))
									(if (>= (count td) 5)
										(list (nth td 4))
										(list nil)))
								td)))))
				/* Normalize to pairs FIRST then flatten — qpp-fields-to-flat's
				match clause '(name expr) accidentally splits `(inner_select sub)`
				into [inner_select, sub] when given flat input (the parser shape).
				pair-normalize is idempotent on already-pair input. */
				(qpp-fields-to-flat (qpp-fields-to-pairs (qpp-tuple-fields collapsed)))
				(qpp-tuple-condition collapsed)
				(qpp-tuple-group collapsed)
				(qpp-tuple-having collapsed)
				(qpp-tuple-order collapsed)
				(qpp-tuple-limit collapsed)
				(qpp-tuple-offset collapsed))))))

(define qpn-restore-root-stage-limits (lambda (tuple root-limit root-offset)
	(if (or (not (nil? root-limit)) (not (nil? root-offset)))
		tuple
		(qpp-rebuild-tuple
			(qpp-tuple-schema tuple)
			(qpp-tuple-tables tuple)
			(qpp-tuple-fields tuple)
			(qpp-tuple-condition tuple)
			(map (coalesceNil (qpp-tuple-group tuple) '()) (lambda (stage)
				(if (or
					(stage_is_dedup stage)
					(not (nil? (stage_partition_aliases stage))))
					stage
					(stage_set
						(stage_set stage (quote limit) nil)
						(quote offset) nil))))
			(qpp-tuple-having tuple)
			(qpp-tuple-order tuple)
			root-limit
			root-offset))))

(define qpn-table-scalar-helper-aliases (lambda (tables)
	(filter (map (coalesceNil tables (list)) (lambda (td)
		(match td
			'(alias_ _ source_ _ _)
			(if (and (qpu-low-scalar-helper-alias? alias_)
				(qpu-low-scan-tagged-table? source_))
				alias_
				nil)
			nil)))
		(lambda (alias_) (not (nil? alias_))))))

(define qpn-push-local-field-guards-for-aliases (lambda (tuple aliases)
	(reduce aliases (lambda (acc alias_)
		(begin
			(define pulled
				(qpu-low-pull-rhs-local-value-guards-fields
					(qpp-fields-to-pairs (qpp-tuple-fields acc)) alias_))
			(define guard-cond (nth pulled 0))
			(if (or (nil? guard-cond) (equal? guard-cond true)
				(equal? guard-cond (quote true)))
				acc
				(qpp-rebuild-tuple
					(qpp-tuple-schema acc)
					(qpu-low-add-join-cond-to-table
						(qpp-tuple-tables acc) alias_ guard-cond)
					(qpp-fields-to-flat (nth pulled 1))
					(qpp-tuple-condition acc)
					(qpp-tuple-group acc)
					(qpp-tuple-having acc)
					(qpp-tuple-order acc)
					(qpp-tuple-limit acc)
					(qpp-tuple-offset acc)))))
		tuple)))

(define qpn-push-local-field-guards (lambda (tuple)
	(if (not (qpp-tuple? tuple)) tuple
		(qpn-push-local-field-guards-for-aliases
			tuple
			(qpn-table-scalar-helper-aliases (qpp-tuple-tables tuple))))))

/* qpn-compile-derived-subqueries — a FROM-derived SELECT is its own SQL
scope. Compile that scope through the same Neumann pipeline before the
enclosing SELECT is lifted, so no inner_select marker can leak through a
preserved derived-table boundary (for example SELECT t.* FROM (SELECT ...)).
This is recursive over nested derived tables. */
(define qpn-compile-derived-subqueries-in-expr (lambda (expr) (match expr
	(cons sym args) (begin
		(define kind (qpl-marker-kind expr))
		(match kind
			'inner_select
			(match args
				'(subquery)
				(list sym (qpn-compile-derived-subqueries subquery))
				(cons sym (map args qpn-compile-derived-subqueries-in-expr)))
			'inner_select_exists
			(match args
				'(subquery)
				(list sym (qpn-compile-derived-subqueries subquery))
				(cons sym (map args qpn-compile-derived-subqueries-in-expr)))
			'inner_select_in
			(match args
				'(target subquery)
				(list sym
					(qpn-compile-derived-subqueries-in-expr target)
					(qpn-compile-derived-subqueries subquery))
				(cons sym (map args qpn-compile-derived-subqueries-in-expr)))
			(cons sym (map args qpn-compile-derived-subqueries-in-expr))))
	expr)))

(define qpn-compile-derived-subqueries (lambda (t)
	(if (not (qpp-tuple? t)) t
		(qpp-rebuild-tuple
			(qpp-tuple-schema t)
			(map (coalesceNil (qpp-tuple-tables t) (list)) (lambda (td)
				(if (or (nil? td) (< (count td) 3)) td
					(begin
						(define tname (nth td 2))
						(if (qpp-tuple? tname)
							(merge (list (nth td 0) (nth td 1)
								(if (qpp-tuple-contains-window? tname)
									(qpn-compile-derived-subqueries tname)
									(neumann_compile_select tname)))
								(if (>= (count td) 4)
									(list (nth td 3))
									(list false))
								(if (>= (count td) 5)
									(list (nth td 4))
									(list nil)))
							td)))))
			(qpp-map-fields (qpp-tuple-fields t) qpn-compile-derived-subqueries-in-expr)
			(qpn-compile-derived-subqueries-in-expr (qpp-tuple-condition t))
			(qpp-map-group (qpp-tuple-group t) qpn-compile-derived-subqueries-in-expr)
			(qpn-compile-derived-subqueries-in-expr (qpp-tuple-having t))
			(qpp-map-order (qpp-tuple-order t) qpn-compile-derived-subqueries-in-expr)
			(qpp-tuple-limit t)
			(qpp-tuple-offset t)))))

/* neumann_compile_select tuple [schemas] →
a clean 7-tuple ready for build_queryplan_inner.

If schemas is nil, column_resolve_pass uses an empty schema list (resolver
returns input expressions unchanged for unknown aliases — safe for the
test-style usage where canonicalization isn't required). Production
callers supply the real schemas list. */
(define neumann_compile_select (lambda (tuple) (begin
	(if (not (qpp-tuple? tuple))
		(error "neumann_compile_select: input is not a 7-tuple") nil)
	/* Parser emits fields as FLAT (name1 expr1 name2 expr2 …) per
	sql-parser.scm sql_select_core's (merge cols). My pipeline operates
	on list-of-pairs ((name1 expr1) (name2 expr2) …) for clarity.
	Convert at entry; convert back at exit. */
	(define tuple-pairs (qpp-rebuild-tuple
		(qpp-tuple-schema tuple)
		(qpp-tuple-tables tuple)
		(qpp-fields-to-pairs (qpp-tuple-fields tuple))
		(qpp-tuple-condition tuple)
		(qpp-tuple-group tuple)
		(qpp-tuple-having tuple)
		(qpp-tuple-order tuple)
		(qpp-tuple-limit tuple)
		(qpp-tuple-offset tuple)))
	/* Pre-pass: disambiguate alias collisions across SQL scopes (FAQ §35).
	v2 preserves field-shape per sub-tuple to avoid corrupting downstream
	legacy consumers that expect flat form (e.g. derived sub-tuples
	carrying window_func expressions). */
	(define t0 (alias_disambiguate_pass tuple-pairs))
	(define t1 (alias_normalize_pass t0))
	/* Use scope-aware column resolution: recurses into inner_select / _in /
	_exists markers with the sub-tuple's own local schemas, so nil-tv refs
	resolve correctly inside nested scopes (per SQL scope rules). Schemas
	come from the live storage via `(show schema table)` per
	qpp-schemas-from-tables — derived sub-tuples expose their projection
	names as columns. */
	(define t1b (qpn-compile-derived-subqueries t1))
	(define t2 (column_resolve_scoped_pass t1b (list)))
	(define t3 (derived_table_inline_pass t2))
	/* Re-resolve after inlining: derived-table inlining can introduce refs
	to the now-directly-visible underlying tables that were nil-tv inside
	the derived sub-tuple. A second scoped resolve binds them with the new
	table set visible. Idempotent for already-resolved refs (no false=false
	markers to resolve). */
	(define t3b (column_resolve_scoped_pass t3 (list)))
	(define t4 (lift_dep_joins_pass t3b))
	(if neumann_pipeline_trace (print "[neumann] lifted: " t4) nil)
	(define t5 (unnest_pass t4))
	(if neumann_pipeline_trace (print "[neumann] unnested: " t5) nil)
	(define t6 (lower_to_scans_pass t5))
	(define t6-root-limits
		(qpn-restore-root-stage-limits
			t6
			(qpp-tuple-limit tuple-pairs)
			(qpp-tuple-offset tuple-pairs)))
	(if (not (qpp-tuple? t6))
		(error "neumann_compile_select: pipeline did not produce a 7-tuple")
		/* Re-flatten fields recursively (including derived sub-tuples) to
		match the flat (name1 expr1 …) parser convention. */
		(qpn-push-local-field-guards
			(qpn-flatten-tuple-recursive t6-root-limits))))))

/* neumann_compile_select_with_schemas tuple schemas →
Same as above but with an explicit schemas list passed to column_resolve. */
(define neumann_compile_select_with_schemas (lambda (tuple schemas) (begin
	(if (not (qpp-tuple? tuple))
		(error "neumann_compile_select_with_schemas: input is not a 7-tuple") nil)
	(define tuple-pairs (qpp-rebuild-tuple
		(qpp-tuple-schema tuple)
		(qpp-tuple-tables tuple)
		(qpp-fields-to-pairs (qpp-tuple-fields tuple))
		(qpp-tuple-condition tuple)
		(qpp-tuple-group tuple)
		(qpp-tuple-having tuple)
		(qpp-tuple-order tuple)
		(qpp-tuple-limit tuple)
		(qpp-tuple-offset tuple)))
	(define t1 (alias_normalize_pass tuple-pairs))
	(define t1b (qpn-compile-derived-subqueries t1))
	(define t2 (column_resolve_scoped_pass t1b schemas))
	(define t3 (derived_table_inline_pass t2))
	(define t3b (column_resolve_scoped_pass t3 schemas))
	(define t4 (lift_dep_joins_pass t3b))
	(define t5 (unnest_pass t4))
	(define t6 (lower_to_scans_pass t5))
	(define t6-root-limits
		(qpn-restore-root-stage-limits
			t6
			(qpp-tuple-limit tuple-pairs)
			(qpp-tuple-offset tuple-pairs)))
	(if (not (qpp-tuple? t6))
		(error "neumann_compile_select_with_schemas: pipeline did not produce a 7-tuple")
		(qpn-push-local-field-guards
			(qpn-flatten-tuple-recursive t6-root-limits))))))

/* ==================== Trace toggle (debugging only) ==================== */

/* neumann_pipeline_trace — when set to true, prints input + lowered tuple
for each query routed through the new pipeline. Default off. */
(set neumann_pipeline_trace false)
