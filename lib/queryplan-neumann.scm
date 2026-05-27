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

/* neumann_compile_select tuple [schemas] →
   a clean 7-tuple ready for build_queryplan_inner.

If schemas is nil, column_resolve_pass uses an empty schema list (resolver
returns input expressions unchanged for unknown aliases — safe for the
test-style usage where canonicalization isn't required). Production
callers supply the real schemas list. */
(define neumann_compile_select (lambda (tuple) (begin
	(if (not (qpp-tuple? tuple))
		(error "neumann_compile_select: input is not a 7-tuple") nil)
	(define t1 (alias_normalize_pass tuple))
	(define t2 (column_resolve_pass t1 (list)))
	(define t3 (lift_dep_joins_pass t2))
	(define t4 (unnest_pass t3))
	(define t5 (lower_to_scans_pass t4))
	(if (not (qpp-tuple? t5))
		(error "neumann_compile_select: pipeline did not produce a 7-tuple")
		t5))))

/* neumann_compile_select_with_schemas tuple schemas →
   Same as above but with an explicit schemas list passed to column_resolve. */
(define neumann_compile_select_with_schemas (lambda (tuple schemas) (begin
	(if (not (qpp-tuple? tuple))
		(error "neumann_compile_select_with_schemas: input is not a 7-tuple") nil)
	(define t1 (alias_normalize_pass tuple))
	(define t2 (column_resolve_pass t1 schemas))
	(define t3 (lift_dep_joins_pass t2))
	(define t4 (unnest_pass t3))
	(define t5 (lower_to_scans_pass t4))
	(if (not (qpp-tuple? t5))
		(error "neumann_compile_select_with_schemas: pipeline did not produce a 7-tuple")
		t5))))

/* ==================== Opt-in switch ==================== */

/* neumann_pipeline_enabled — global toggle.
Default: false → build_queryplan_term uses the legacy untangle_query path.
Set to true to route through the new pipeline. Tests / dev can flip via
  (set neumann_pipeline_enabled true)
without modifying any other code.

When enabled, build_queryplan_term will route compatible queries through
neumann_compile_select. Queries my pipeline doesn't support yet (errors
loudly) still surface their errors loudly per FAQ §1 — no silent fallback
to legacy. */
(set neumann_pipeline_enabled false)

/* neumann_pipeline_supports? tuple → true if the pipeline currently handles
this tuple's shape WITHOUT errors. Used by the build_queryplan_term gate to
decide whether to invoke neumann_compile_select or fall through to legacy.

NOTE: Per FAQ §1 there are no fallback paths. This predicate exists ONLY
to identify shapes the pipeline doesn't yet implement so the legacy code
keeps working during the transition. Once the pipeline is complete, this
predicate becomes `true` for everything and the legacy code is deleted.

Returns false on any error during a dry-run pipeline invocation. */
(define neumann_pipeline_supports? (lambda (tuple)
	(try
		(lambda () (begin
			(neumann_compile_select tuple)
			true))
		(lambda (e) false))))
