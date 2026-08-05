# Copyright (C) 2026  Carl-Philip Haensch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

# MemCP Query Planner Invariants

This document is the architecture contract for MemCP query planning work.
Branches may improve the implementation, add physical operators, or change
internal helper encodings, but they must preserve the semantic and phase
boundaries described here.

These invariants apply especially to `lib/queryplan.scm`, `lib/sql-parser.scm`,
`lib/psql-parser.scm`, and physical scan/storage lowering code.

## Planner Pipeline

The planner pipeline is:

```text
parser AST
-> untangle_query
-> join_reorder / optimize
-> build_queryplan
```

Each phase has a distinct job:

- parser: preserve SQL syntax and produce neutral AST/query terms
- `untangle_query`: build a decorrelated logical IR
- `join_reorder` / optimize: choose logical order and record costing facts
- `build_queryplan`: choose physical carriers/operators and emit Scheme code

Do not move physical decisions into `untangle_query`. Do not leave logical
decorrelation work for `build_queryplan`.

## Logical IR Shape

The logical planner is based on three combined operator shapes:

- `query-block`: select/join/filter/project/order/limit work unit
- `group-stage`: domain, aggregate, EXISTS/IN/scalar helper, HAVING, and window
  partition aggregate work unit
- `union-block`: set-operation work unit

`stage-output` is a logical relation descriptor produced by `group-stage` or
`union-block`. It is not a fourth algebra operator.

This coarse model is intentional. MemCP should not split the logical layer into
many textbook operators such as separate scan/select/project/join/order nodes.
That creates artificial boundaries, expensive rewrite churn, and many later
special cases. The physical engine already has combined operators; the logical
IR should preserve enough semantic structure while remaining coarse enough to
lower efficiently.

There is no logical `scan` operator. `scan`, `scan_order`,
`scan_order_multi`, keytables, ORC columns, RecSets, temp columns, and direct
nested scans are physical lowering choices.

## Logical vs Physical Boundary

After `untangle_query`, the IR must not contain physical artifacts:

- no `scan` / `scan_order` logical operators
- no `.grp:*` physical helper relation names
- no ORC/temp column physical carrier names
- no RecSet physical values
- no storage-specific helper relation names

The logical IR may contain semantic facts that make these physical choices
possible later, for example:

- `purpose`
- `domain`
- `lookup_keys`
- `null_semantics`
- `cardinality_mode`
- `preserve_empty_domain`
- order/window/group facts

`build_queryplan` chooses the physical implementation after decorrelation and
reordering. The same logical helper may lower to a group cache, direct scan,
`scan_exists`, RecSet, ORC, `scan_order_multi`, or temp table depending on
costs and semantics.

## No Subquery Fallback

Every supported correlated subquery shape must be lowered through relational
decorrelation.

Forbidden after `untangle_query`:

- `inner_select*`
- `neumann_*` marker soup
- `dependent-subquery`
- correlated scalar/EXISTS/IN expressions
- recursive scalar execution paths in expression lowering

If an operator case is missing, the implementation must raise an explicit
unsupported-case error. It must not execute the subquery recursively as a
fallback.

Physical execution may still contain helper queries, prepared scans, nested
scans, group-cache fills, or RecSet construction. That is valid only after all
logical dependencies have been converted into explicit domain keys, lookup
keys, stage outputs, and joins.

## Neumann / Top-Down Decorrelation

MemCP follows the Neumann/NK15/BTW2025 decorrelation model described in:

- `papers/Unnesting-Arbitrary-Queries.pdf`
- `papers/neumann-improving-unnesting-btw2025.pdf`

Every dependent subquery first goes through simple unnesting:

- collect equality classes (`cclasses`)
- derive representative substitutions (`repr`)
- pull predicates/maps when valid
- convert trivial dependent joins into normal joins

Only if an accessing operator still depends on outer references do we build
general Domain D. Domain D is the fallback after simple unnesting fails, not the
default for every correlation.

Nested dependent joins must share a parent-chained top-down context for outer
refs, cclasses, repr/substitution maps, domain keys, and shared roots. Do not
restart full bottom-up analysis for every inner subquery.

## Domain D

Domain D is the duplicate-free projection of all outer references that the
inner side actually reads.

It must not contain duplicate bindings. Duplicate domain rows break per-key
outer/miss/anti semantics and can duplicate results.

Session reads used inside dependent helpers are treated like outer dependencies
when they affect semantics. Pull their values into the helper domain/key
context and join back with null-safe equality when needed.

## Scalar Cardinality

Scalar cardinality is semantic, not a physical operator.

The logical stage must distinguish:

- `many`: normal non-scalar relation
- `first`: explicit SQL `LIMIT 1`; first matching row wins, zero rows yield
  `NULL`
- `single_or_error`: scalar subquery without explicit `LIMIT 1`; zero rows
  yield `NULL`, one row yields the value, and the second matching row per
  domain binding throws the "more than one row" error

Physical lowering may implement `single_or_error` with `scan_order` partition
limits, `LIMIT 2`, reducers, promise/session state, or another efficient
mechanism. It must never silently degrade to `LIMIT 1`.

## EXISTS, IN, and NOT IN

EXISTS is modeled as presence semantics over a domain, usually via a
`group-stage` plus `stage-output` lookup.

Required logical facts include:

- `purpose = exists` or `not_exists`
- `domain`
- `lookup_keys`
- `cardinality_mode = many`
- `null_semantics = exists`
- `preserve_empty_domain` when misses must preserve every outer key

IN is membership semantics, not a raw EXISTS clone. IN and NOT IN must preserve
SQL three-valued NULL behavior. A membership stage must distinguish at least:

- matching RHS value exists
- RHS contains NULL
- probe expression is NULL

`NOT IN` must not be rewritten through presence-only logic. UNION rewrites
inside IN context are allowed only after duplicates and NULL facts remain
correct for the membership context.

## Groups and HAVING

Groups are hard logical stage boundaries.

Aggregate markers are collected into `group-stage` definitions. The outer query
reads aggregate, presence, scalar, and HAVING-filtered results through
`stage-output` relations.

HAVING is post-group semantics. It belongs to the `group-stage`, not to the
input-row WHERE. It must not filter input rows before aggregation unless a rule
proves that key-only post-group pruning is semantically identical.

Empty correlated aggregate groups must preserve one result per required domain
binding:

- COUNT returns 0
- SUM/MIN/MAX/AVG and most other aggregates return SQL `NULL` where applicable
- missing inner rows must not erase the outer binding

## Derived Tables and Materialization

Ordinary `FROM (SELECT ...)` must be inlined early by `untangle_query`.

Inlining builds a renaming map, prunes unused visible and hidden slots, and
merges predicates after renaming. Ordinary derived-table materialization is not
a default strategy.

Materialize only when a real semantic or physical barrier exists:

- reusable group/aggregate cache
- conflicting window order requiring ORC
- shared CTE/DAG root
- explicit materialization semantics
- physical cost model chooses a query-local carrier such as RecSet
- temp table as last-resort physical barrier

## UNION

Logical UNION stays as `union-block`.

Physical lowering preferences:

- unordered `UNION ALL`: successive branch emits
- unordered `UNION`: dedup barrier
- ordered `UNION ALL`: `scan_order_multi` when branches are streamable
- non-streamable ordered union: narrow materialization only when necessary

Do not collapse UNION branches into opaque helpers that hide unnesting,
membership, dedup, or NULL semantics from later planner phases.

## Windows

A window is not automatically an ORC/materialization barrier.

If one scan order can satisfy the base query and all window partitions/orders,
lower as a fused ordered scan. Use ORC/window-stage only when orders conflict,
the window result is shared, or the value must survive as a computed column.

Correlated window functions must include the outer domain keys in their
partitioning. Otherwise different outer bindings mix into the same window.

## Cost Model and Physical Lowering

Lowering decisions must become cost-aware and explicit. Do not add scattered
special-case branches when the decision is really "which physical carrier is
cheapest for this logical stage?"

The lowerer/cost model may consider:

- table and stage cardinality
- filter selectivity
- distinct domain/key counts
- order/limit compatibility
- warm/cold state of reusable carriers
- expected reuse count
- build cost vs probe cost
- memory footprint
- compile-time cost
- index/auto-index availability
- telemetry from previous scans or carrier builds

Physical options include:

- reusable group keytable plus computed aggregate columns
- FK-backed cached column
- direct one-pass group scan
- `scan_exists`
- RecSet or RecSet projection
- `scan_order` / `scan_order_multi`
- ORC computed column
- temp table as a last resort

The selected physical shape should be visible in EXPLAIN when it matters for
review or regression protection.

## Canonical Naming and Reuse

Helper identities must be canonical.

Derive physical helper names from:

- source identity
- full normalized key set
- canonical filter/condition forms when they affect the carrier result
- semantic facts that change the result

Never derive helper identity from visible aliases, wrapper-local aliases, or
projection titles.

The same keytable plus the same aggregate formula should produce the same
computed column. Prefer reusable full key domains over many narrow one-off
keytables when additional filters can be applied while reading stage output.

## Tests and Review Gates

A branch touching query planning is not ready merely because tests pass.

It must also preserve architecture and performance:

- no fallback markers survive `untangle_query`
- no physical helper relations leak before `build_queryplan`
- scalar cardinality tests cover zero, one, and two rows per domain
- IN/NOT IN NULL cases are correct or explicitly unsupported
- HAVING tests prove post-group filtering
- EXPLAIN/EXPLAIN IR tests protect important plan shapes
- performance claims include A/B measurements against current master
- representative long-running queries are not slower than master
- compile-time and plan-size guards cover nested/decorrelation-heavy cases

Do not weaken tests, mark critical tests noncritical, or replace performance
guards with shape-only assertions unless the lost protection is restored
elsewhere.
