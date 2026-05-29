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

This module exposes the public API. Wiring into build_queryplan_term is
gated by `(neumann_pipeline_enabled)` so the default behavior is preserved
during development.
*/

/* ==================== Public API ==================== */

/* qpn-flatten-tuple-recursive — convert all fields lists in a 7-tuple from
list-of-pairs to flat (name1 expr1 name2 expr2 …), INCLUDING any derived
sub-tuples nested in the tables list. The legacy build_queryplan_inner /
untangle_query consumers iterate fields via extract_assoc / reduce_assoc
which assume the flat form. */
(define qpn-flatten-tuple-recursive (lambda (t)
	(if (not (qpp-tuple? t)) t
		(qpp-rebuild-tuple
			(qpp-tuple-schema t)
			(map (coalesceNil (qpp-tuple-tables t) (list)) (lambda (td)
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
			(qpp-fields-to-flat (qpp-fields-to-pairs (qpp-tuple-fields t)))
			(qpp-tuple-condition t)
			(qpp-tuple-group t)
			(qpp-tuple-having t)
			(qpp-tuple-order t)
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
	(define t1 (alias_normalize_pass tuple-pairs))
	/* Use scope-aware column resolution: recurses into inner_select / _in /
	   _exists markers with the sub-tuple's own local schemas, so nil-tv refs
	   resolve correctly inside nested scopes (per SQL scope rules). Schemas
	   come from the live storage via `(show schema table)` per
	   qpp-schemas-from-tables — derived sub-tuples expose their projection
	   names as columns. */
	(define t2 (column_resolve_scoped_pass t1 (list)))
	(define t3 (derived_table_inline_pass t2))
	/* Re-resolve after inlining: derived-table inlining can introduce refs
	   to the now-directly-visible underlying tables that were nil-tv inside
	   the derived sub-tuple. A second scoped resolve binds them with the new
	   table set visible. Idempotent for already-resolved refs (no false=false
	   markers to resolve). */
	(define t3b (column_resolve_scoped_pass t3 (list)))
	(define t4 (lift_dep_joins_pass t3b))
	(define t5 (unnest_pass t4))
	(define t6 (lower_to_scans_pass t5))
	(if (not (qpp-tuple? t6))
		(error "neumann_compile_select: pipeline did not produce a 7-tuple")
		/* Re-flatten fields recursively (including derived sub-tuples) to
		   match the flat (name1 expr1 …) parser convention. */
		(qpn-flatten-tuple-recursive t6)))))

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
	(define t2 (column_resolve_scoped_pass t1 schemas))
	(define t3 (derived_table_inline_pass t2))
	(define t3b (column_resolve_scoped_pass t3 schemas))
	(define t4 (lift_dep_joins_pass t3b))
	(define t5 (unnest_pass t4))
	(define t6 (lower_to_scans_pass t5))
	(if (not (qpp-tuple? t6))
		(error "neumann_compile_select_with_schemas: pipeline did not produce a 7-tuple")
		(qpp-rebuild-tuple
			(qpp-tuple-schema t6)
			(qpp-tuple-tables t6)
			(qpp-fields-to-flat (qpp-tuple-fields t6))
			(qpp-tuple-condition t6)
			(qpp-tuple-group t6)
			(qpp-tuple-having t6)
			(qpp-tuple-order t6)
			(qpp-tuple-limit t6)
			(qpp-tuple-offset t6))))))

/* ==================== Trace toggle (debugging only) ==================== */

/* neumann_pipeline_trace — when set to true, prints input + lowered tuple
for each query routed through the new pipeline. Default off. */
(set neumann_pipeline_trace false)
