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
- `build_queryplan`: choose physical scan sources/operators and emit Scheme code

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
- no physical intermediate-relation names such as ORC/temp columns
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

Positive membership in a WHERE truth context may use the semantic
`membership_truth` expression primitive. It references a logical group-stage
output and is not a physical RecSet or scan. Canonicalization may flatten
`IN (… UNION …)` to a n-ary OR of these primitives because SQL `UNKNOWN` and
`FALSE` both reject the row in that context. This rewrite must not cross `NOT`,
`CASE`, projection, or another value-producing context; those retain the full
match/RHS-NULL representation required for three-valued logic.

### RecSet algebra as an optimization proof system

RecSets provide a general way to discover and justify physical formula
rewrites. For a fixed base relation `R`, define `T_R(p)` as the exact set of
record IDs for which predicate `p` is SQL `TRUE`. In a truth-filtering context:

```
T_R(false)   = empty
T_R(true)    = all visible records of R
T_R(p OR q)  = T_R(p) union T_R(q)
T_R(p AND q) = T_R(p) intersect T_R(q)
p implies q  => T_R(p) is a subset of T_R(q)
```

Ordinary set identities such as associativity, commutativity, distributivity,
absorption, and factoring may therefore be used by planner developers to find
and prove useful normalizations. For example,
`(A intersect B) union (A intersect C) = A intersect (B union C)` can expose one
shared selective scan instead of two. Once such a transformation is proved, it
belongs as a recognized pattern in a normalization pass over semantic
primitives. The optimizer is not required to search arbitrary set formulas at
query-compilation time. This is a general method for extending the optimizer,
not a rule tied to one SQL spelling and not a mandate to execute a RecSet.

A physical candidate RecSet need not always be exact. If `C_R(p)` is only a
proven superset of `T_R(p)`, it is a safe scan boundary only while the original
predicate `p` remains as a residual filter. This permits cheap projections or
partially extracted branches to reduce the search space without claiming a
stronger equivalence than was proved. Union/intersection operands must refer to
the same base relation and visibility snapshot.

Containment proofs compose monotonically:

```
C_R(p) contains T_R(p), C_R(q) contains T_R(q)
  => C_R(p) union C_R(q) contains T_R(p OR q)
  => C_R(p) intersect C_R(q) contains T_R(p AND q)
```

Every RecSet rewrite must therefore record or establish whether its result is
exact or merely a candidate, identify its base relation and snapshot, and keep
the residual predicate unless exactness has been proved. Formula equality is
not by itself permission to change result multiplicity, ordering, NULL
extension, or LIMIT boundaries; RecSets prove which base records may be scanned,
not how often or in which semantic order result rows are produced.

Semijoins extend the same algebra across relations. If `S` is an exact or safe
candidate RecSet on a source relation, projecting `S` through join keys to `R`
produces the corresponding exact or candidate target RecSet. Duplicate source
keys do not matter because RecSets contain record IDs, not result multiplicity.
This is why membership, EXISTS, joins, and boolean filter clouds can share
physical RecSet optimization even when their parser ASTs differ.

As one instance of the general rules, for probe expression `x` and UNION
branches `Q_i`, positive membership permits:

```
T_R(x IN (Q_1 UNION ... UNION Q_n))
  = union(T_R(x IN Q_i) for each i)
  = T_R(EXISTS(Q_1 matching x) OR ... OR EXISTS(Q_n matching x))
```

That example is not the axiom itself. New syntactic corner cases should
normalize to semantic primitives. Developers may use RecSet algebra to derive
and prove further normalization patterns; the common lowerer must still choose
between ordinary scans with indexed probes, projected RecSets, driver order,
and braking from statistics. A normalization justified using truth sets must
never preselect RecSet execution.

Complement and difference require extra care under SQL three-valued logic:
`T_R(NOT p)` is not generally the complement of `T_R(p)` because `UNKNOWN` is
in neither truth set. Such rewrites require a proven two-valued predicate or an
explicit representation of FALSE and UNKNOWN. Consequently these identities
do not apply unchanged to `NOT IN`, `NOT`, or value-producing SQL-3VL contexts.

`build_queryplan` chooses the physical implementation after decorrelation and
reordering. The same logical helper may lower to a group cache, direct scan,
`scan_exists`, RecSet, ORC, `scan_order_multi`, or temp table depending on
costs and semantics.

## Join Order Has A Single Owner

Join ordering is a logical optimization. Once `join_reorder` has selected the
driver and produced a join tree for a join cloud, physical lowering must
preserve the leaf traversal encoded by that tree when it emits nested scans.
The query-block source list is an immutable semantic alias-to-source catalog
after logical planning. Physical preparation must never reorder, replace, or
rewrite that list into a second representation of the chosen plan.

Physical lowering must consume the join tree structurally. A bushy tree must
not be flattened into a leaf sequence for scan lowering, predicate placement,
or null handling. Every physical step must retain the current join node's left
and right subtrees, join kind, bound-alias set, and null-extension boundary.
Helpers may derive local facts from a subtree, but they must not discard its
boundaries or make another driver/order decision.

The tree also preserves execution independence. Physical lowering decides per
node whether independent child subtrees run through `(parallel)`, serial
`(list)`, or dependency-driven nested scans. That physical choice may use cost
and batch-size information, but it must not serialize independent branches by
first reducing the tree to one global leaf order.

Physical lowering still chooses the operator and scan source for each tree node.
Depending on the cost and semantics, a source may become a
direct subscan, group keytable, computed-column-filtered slice of a source
keytable, RecSet, ORC column, or another storage-engine scan source. Operator choice
must not change the logical join tree's leaf order.

## Guarded Query-Plan Specialization

The Scheme query-plan cache may hold several lazily compiled physical plans for
one normalized SELECT shape. DDL, DML, and transaction control retain exact
cache entries because their formulas may intentionally operate on the request
session. This is a polymorphic cache entry, not a second planner and not a
storage-engine query cache.

Only SELECT shapes which reach a parameter- or statistics-dependent planner
decision use polymorphic entries. Shapes without a tipping decision retain the
smaller exact cache path.

Every parameter- or statistics-dependent cost-model tipping decision must
record the executable condition under which its chosen alternative remains
valid. During one compilation these conditions are accumulated in a dedicated
`newsession`; this compile-local state substitutes for threading a writer monad
through the functional planner. It must not escape as logical or physical IR.
The compile session must not capture request-local execution state such as
`__memcp_tx`; cached scans resolve that state from the executing session.

The conjunction of the recorded conditions guards the specialized plan. A
guard miss compiles exactly one plan for the current bindings and statistics,
then prepends its condition/plan pair to the variadic Scheme `if` cache entry.
New variants go first because the newest statistics regime is normally the
most likely one. Existing immutable plan tails remain valid for in-flight
queries.

Guards repeat cost decisions, not query planning. They must be side-effect-free
and must not scan relations, build indexes or group caches, acquire schema
write locks, or materialize data. Repeated catalog inputs must be bound once
per guard evaluation. If a useful generalized inequality is unavailable, an
exact parameter/statistics-input guard is the conservative fallback; an old
specialized plan must never be selected after an unguarded cost input changes.

The cache value must keep the complete Scheme guard/plan formula visible to the
existing cachemap size accounting and eviction mechanism. A `newsession` may
hold the entry's compile lock and metadata, but it must not hide the plan tree
as the cachemap's sole value.

## Relational Results Stay In The Storage Engine

A **scan source** is a storage-engine representation that `scan`, `scan_order`,
or another physical scan operator can consume. Base tables, RecSets, group
keytables, and query-local temporary tables are scan sources. An
**intermediate relation** is a relational result materialized in such a scan
source between physical plan stages. A **group cache** is a reusable
intermediate relation for grouped keys and aggregates; its default physical
representation is a **group keytable**. Use these specific terms rather than
an ambiguous umbrella term.

Base and intermediate relational data already lives in the in-memory storage
engine. Execution must consume it through `scan`, `scan_order`,
`scan_order_multi`, RecSets, ORC columns, group keytables, or another explicitly
selected storage-engine scan source. It must not copy a relational result into a
Scheme list, association list, FastDict, or query-local memo merely to join,
order, limit, or probe it.

Small planner metadata, scalar runtime values, and aggregate state may use
Scheme data structures. A physical scan source must remain the single relational
representation of its result: choosing a direct/grouped subscan must not also
build the complete keytable, and choosing a keytable must not duplicate it into
a Scheme-side relation. A requested subset of an existing source keytable may
be evaluated through computed-column filters instead of constructing another
intermediate relation.

List-backed relations do not provide indexes, statistics, range braking,
late materialization, bounded memory, spill, incremental maintenance, or the
storage engine's batch and concurrency behavior. They therefore cannot be a
cost-based alternative, even when they benchmark faster for small inputs.

Use the existing scalable physical representations instead:

- fused `scan` / `scan_order` pipelines for streamable work
- `scan_order_multi` for ordered UNION inputs
- RecSets for query-local membership
- canonical keytables for groups and aggregate domains
- ORC/temp columns for reusable ordered computation
- canonically named query-local temp tables for unavoidable relational barriers

Lists remain valid for bounded scalar tuples, operator arguments, parser data,
and final API row values. Their size must be independent of relation cardinality.

## Computed Columns Are Definitions, Not Prepared Subsets

A computed column, including a `StorageComputeProxy`, defines one logical value
for every live row in its table. Its physical cache may be complete, partial,
invalidated, or absent without changing that logical domain.

`filter` / `filterCols` arguments and `CompressFiltered` are preparation hints.
They may select values to prewarm because a physical plan expects to read them
soon, but they must never become part of the column definition. A row omitted
from a preparation batch is not absent, `NULL`, or an error.

An ordinary computed column may repair a missing or invalidated value by
computing that row from its current inputs and caching the result. An ordered
reduction column (ORC) has dependencies between rows and must instead repair a
semantically sufficient ordered dependency range. That range may be a suffix
or the whole column. It must not pretend that an order-dependent value can be
computed pointwise, return stale data, or expose a transient invalid-cache
sentinel as a SQL value.

Invalidation marks physical cache entries as unusable; it does not remove their
logical values. Invalidation and repair must cover the dependency closure of
the changed inputs. Planner setup, DDL, rebuild, persistence, and session-local
variants must preserve this definition/cache distinction. Eager preparation,
selective prewarming, and lazy repair may change when work happens, never the
query-visible result.

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

Decorrelated helpers remain nested scans or independently prepared intermediate
relations. They must not replace the limited root scan with a complete lookup
table. Projection-only helpers belong in the limited scan's map callback so
rows rejected by filtering, ordering, offset, or limit never invoke them.

## Predicate Placement In Join Clouds

Physical lowering partitions the complete ON/WHERE predicate across the scans
of the already ordered join cloud. Each conjunct belongs in the scan filter at
the point where its referenced bindings and its required outer-join semantics
are available. This is not simply an "innermost possible" rule: the correct
scan is determined by the bindings and null-extension boundary of that term.

Reordering may annotate an inner join leaf with a single-alias predicate that
it has already classified and costed. This records semantic alias ownership so
the physical lowerer does not have to rediscover it. A `WHERE` predicate on a
nullable outer-join side instead belongs to the barrier node, because its NULL
semantics are available only after null extension. The predicate remains in the
query-block condition as the authoritative logical expression; the annotation
must neither remove it nor introduce a physical `scan`. Physical lowering
consumes the annotation exactly once when it chooses the scan operator and
access path. Outer-join `ON` predicates also stay on their barrier-owning join
node and must never be converted into leaf annotations.

Every conjunct must appear exactly once at a semantically valid scan or remain
as an explicit residual predicate. Predicate placement must never weaken or
drop a join condition, turn an equijoin into a cross product, or evaluate a
nullable-side WHERE term as though it were an ON term.

Predicate lowering operates on individually tracked conjuncts. Every conjunct
retains one semantic origin: either the query block's `WHERE` clause or the
specific join barrier whose `ON` clause owns it. Normalized inner-join `ON`
conjuncts may join the common predicate cloud; outer-join `ON` conjuncts remain
owned by their null-extension barrier. Each conjunct is assigned exactly once
to a semantically valid scan or join node, or remains exactly once as an
explicit residual predicate. Lowering must be able to account for every input
conjunct after placement.

Predicate readiness is defined relative to a join-tree node, not a flat source
suffix. An inner predicate may run as soon as all referenced bindings exist. An
outer-join `ON` predicate filters the nullable input before null extension. A
`WHERE` predicate observing that nullable input runs only after the owning join
node has produced either a matching row or its synthetic NULL row.

## Planner Cache And Group Cache

The query-plan cache and group caches are separate concepts with different
ownership and lifetimes.

- The query-plan cache is Scheme-scoped planner state. It does not create a new
  storage-engine catalog or persistent schema objects.
- A group cache is a reusable storage-engine relation. It remains registered in
  the persistent schema catalog so later queries can reuse it until cache
  eviction removes it according to the existing lifecycle contract.

Do not introduce a storage-level query cache, move group caches into transient
planner state, or merge the two lifecycles merely to simplify physical
lowering or lock management.

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

Intermediate-relation projection is valid only when all semantic inputs are represented:

- every key has a matching projected driver column or an explicit external
  binding
- the full local filter remains attached to the intermediate-relation build or as a residual
  predicate
- session-dependent intermediate relations are query-local and cannot be reused across
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
- physical cost model chooses a query-local scan source such as RecSet
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
special-case branches when the decision is really "which physical scan source
or intermediate relation is cheapest for this logical stage?"

The lowerer/cost model may consider:

- table and stage cardinality
- filter selectivity
- distinct domain/key counts
- order/limit compatibility
- warm/cold state of reusable group caches and intermediate relations
- expected reuse count
- build cost vs probe cost
- memory footprint
- compile-time cost
- index/auto-index availability
- telemetry from previous scans or intermediate-relation builds

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
- canonical filter/condition forms when they affect the cached or materialized result
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
