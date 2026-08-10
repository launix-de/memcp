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

## Join Order Has A Single Owner

Join ordering is a logical optimization. Once `join_reorder` has selected the
driver and ordered the dependent sources in a join cloud, physical lowering
must preserve that order when it emits nested scans. It must not choose another
driver, split the sources into a different execution order, or otherwise make
a second join-order decision.

Physical lowering still chooses the operator and carrier for each source in
that fixed order. Depending on the cost and semantics, a source may become a
direct subscan, group keytable, computed-column-filtered slice of a source
keytable, RecSet, ORC column, or another storage-engine carrier. Operator choice
must not change source order.

## Relational Results Stay In The Storage Engine

Base and intermediate relational data already lives in the in-memory storage
engine. Execution must consume it through `scan`, `scan_order`,
`scan_order_multi`, RecSets, ORC columns, group keytables, or another explicitly
selected storage-engine carrier. It must not copy a relational result into a
Scheme list, association list, FastDict, or query-local memo merely to join,
order, limit, or probe it.

Small planner metadata, scalar runtime values, and aggregate state may use
Scheme data structures. A physical carrier must remain the single relational
representation of its result: choosing a direct/grouped subscan must not also
build the complete keytable, and choosing a keytable must not duplicate it into
a Scheme-side relation. A requested subset of an existing source keytable may
be evaluated through computed-column filters instead of constructing another
carrier.

## LIMIT Is A Scan Boundary

Every physical operator that owns a SQL `LIMIT` must lower to `scan_order` or
`scan_order_multi` with that limit. This also applies without an explicit
`ORDER BY`, using an empty sort specification. A limited relational result must
never be copied into a Scheme list or association structure for a later
`sort`/`slice` step.

`scan_order` may also be used when row order is semantically irrelevant. In
that mode it still owns OFFSET/LIMIT and may use top-k pruning: it need not
filter or materialize every qualifying row before selecting the required
number of rows.

Decorrelated helpers remain nested scans or independently prepared relational
carriers. They must not replace the limited root scan with a complete lookup
table. Projection-only helpers belong in the limited scan's map callback so
rows rejected by filtering, ordering, offset, or limit never invoke them.

## Predicate Placement In Join Clouds

Physical lowering partitions the complete ON/WHERE predicate across the scans
of the already ordered join cloud. Each conjunct belongs in the scan filter at
the point where its referenced bindings and its required outer-join semantics
are available. This is not simply an "innermost possible" rule: the correct
scan is determined by the bindings and null-extension boundary of that term.

Every conjunct must appear exactly once at a semantically valid scan or remain
as an explicit residual predicate. Predicate placement must never weaken or
drop a join condition, turn an equijoin into a cross product, or evaluate a
nullable-side WHERE term as though it were an ON term.

## Functional Plans, Optimizer-Owned Mutation

The query planner emits the functional operator API. It must not select `_mut`
variants to force an execution strategy. Replacing a functional operation with
its mutable implementation is exclusively an optimizer decision based on
proven ownership and lifetime information.

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

The internal transaction session `__memcp_tx` identifies execution state, not
SQL input, and must not become a semantic domain key. All other session reads
that can change a helper result are external dependencies, including values
used below grouped or derived query blocks.

## Predicate and Rewrite Closure

Planner rewrites must preserve the complete predicate. A rule may split a
top-level AND into independently placeable terms, but selecting one useful term
does not permit the remaining terms to be dropped. Terms that cannot move must
remain as residual predicates at a semantically valid level.

When a rewrite removes or replaces a stage-output source, it must rewrite every
consumer of that source consistently: source joins, WHERE, projections, GROUP
BY, HAVING, ORDER BY, and hidden fields. Driver/domain keys must be selected
from the rewritten source set, after inherited scalar probes and correlation
bindings have been resolved. Choosing a key from the pre-rewrite source set can
leave free aliases in the physical plan.

Carrier projection is valid only when all semantic inputs are represented:

- every key has a matching projected driver column or an explicit external
  binding
- the full local filter remains attached to the carrier build or as a residual
  predicate
- session-dependent carriers are query-local and cannot be reused across
  different session bindings
- NULL and empty-domain behavior remains that of the logical stage

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

For a scalar cardinality check, top-k of two rows is sufficient and should be
used whenever the chosen scan supports it: zero rows produces `NULL`, one row
produces the scalar value, and observing the second row proves the
more-than-one-row error. The operator need not scan the remaining matches.

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

## Planner Source Layout

Source files should follow the planner phases rather than feature history. A
future split of `lib/queryplan.scm` should keep one-way dependencies in pipeline
order: shared logical IR, decorrelation and logical rewrites, reorder/cost facts,
physical lowering, then parser-facing adapters. Logical modules must not import
or call physical lowering modules.

Do not combine a large mechanical file move with a semantic planner fix. Move
stable phase ranges in a dedicated refactoring PR so review can distinguish
behavior changes from relocation and A/B compile-time measurements remain
meaningful.
