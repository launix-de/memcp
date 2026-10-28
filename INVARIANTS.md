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

# MemCP Planner and Storage Concurrency Invariants

This document is the architecture contract for MemCP query planning and storage
concurrency work.
Branches may improve the implementation, add physical operators, or change
internal helper encodings, but they must preserve the semantic and phase
boundaries described here.

These invariants apply especially to the `lib/queryplan-*.scm` modules loaded
by `lib/queryplan.scm`, `lib/sql-parser.scm`, `lib/psql-parser.scm`, and physical
scan/storage lowering code.

Each invariant is both a software contract and a guarantee for its usage.
Every implementer MUST fulfill the invariant, e.g. if an invariant says that main_count is constant, no one is allowed to change the value after the shard is published. On the other side, everyone can trust the invariant to be fulfilled. So in this example, no code must ever check if main_count has changed.
This keeps code short and correct but also expects discipline from implementers: they must fulfill the invariant.

If existing code violates an invariant, notify the user and fix the violation at its owning abstraction. Do not compensate by adding defensive checks, synchronization or fallback paths to its consumers.

Another strategy to keep code compact and correct is to not view a problem as a set of hundrets of corner cases but instead design common interfaces and management structures that centralize code at one point. An example is the cluster handle interface which can be used for any network-addressable object that can receive messages. This is also the reason why we introduced scm as a helper scripting languages instead of implementing hundrets of specialized operators.

## Common Taxonomy
 - SCM - the Scheme language with a specific dialect optimized for serializable network-sharable lambda routines, focus on parallelization, outer scopes are not mutable from inner scopes. Data transfer between layers happen through storage engine operations or network-capable sync primitives like session (a threadsafe map) and so on.
 - Storage Engine: the implementation of tables, shards and clustering in storage/ (implemented in go)
 - Frontend: a connection method and language that you can talk to memcp, mostly implemented in lib/. Example frontends: RDF, admin SCM, MySQL-protocol, TSQL, Postgres-compatible psql over REST, mysql syntax over REST, custom frontends (e.g. microservices written in scm using the storage engine directly)
 - Storage Backend: a driver for a storage type like filesystem, S3 or Ceph that offers functions to read and write schemas, read, write and remove shards and write or replay logfiles
 - Query Plan: a machine generated piece of scm code that executes a query (e.g. SQL or RDF). It uses the exposed Storage Engine functions like scan aswell as the "resultrow" handle for output
 - Dashboard: a route inside of the http endpoint that offers a user friendly interface to see the server's health status, browse the stored data, dig into storage formats and perform admin tasks like settings, kill processes, run administrative queries (similar to phpmyadmin)
 - Cluster: A bunch of memcp nodes that trust each other and work together on queries when the working set does not fit into the RAM of one single machine. A client can connect to any node and perform the queries there.
 - Replication Slave: a memcp instance that connects to another memcp node as Replication Master, logs in as one or more database users and replicates the data from one or more databases in his own Storage Backend. The goal of replication is to provide full offline read+write access on the edge. The data is synchronized back to the Master so the Master has the role of a backup. Other users can connect to the Master, too and perform SQL queries on the replicated databases aswell; however after 15min at latest, the replication slave will reclaim the data so offline functionality is still guaranteed.
 - MOSI: synchronization protocol to provide race-free synchronized safe access on data that is shared over a cluster or over replication. basic states: M=modified or exclusive write access, O=readonly reader and main coordinator, S=shared, a replica of the same item like an O item but depending on the owner who can revoke the shared state, I=invalid, the item is not loaded on this node; there are substates and variants for technical reasons to prevent double-requesting an item (e.g. states for state transitions)
 - Database - basic unit of a separated pool of tables, each database can have a different Storage Backend
 - Schema - the declaration of tables, columns+types, triggers, views etc. of a database
 - Table - A set of rows and columns, subdivided into shards, may have a partitioning schema
 - Shard - A set of rows (~60k rows, ~250k RAM on average) that is the basic unit for parallelization. There will be no sub-shard parallelization; Each task on a shard may finish in ~<1ms. Big tables consist of multiple shards so work can be parallelized. The coordination overhead with these boundaries is <1% of execution time

## ENGINE Durability Contract

| Engine  | Durability guarantee | Crash recovery | Notes |
|---------|----------------------|----------------|-------|
| `safe`  | Full ACID, power-outage safe | WAL fsync'd + replayed at startup | **Default.** Use for all production data. |
| `logged`| Process-crash safe (no fsync) | WAL written (no fsync) + replayed | Use when fsync latency is unacceptable and hardware power protection is provided. |
| `sloppy`| Columnar files on disk, **no WAL** | In-memory deltas since last rebuild are **lost** | Use only for reconstructible data or caches. |
| `memory`| None — RAM only | **All data lost on restart** | Never for data that must survive a restart. |
| `cache` | rebuildable data | data is rebuildable by OnInit, may be persisted or not | Used for aggregation caches or temptables or materialized views |

## Cluster execution

These rules describe the target architecture, never implement mechanisms that workaround or violate the architecture. Keep implementation central and minimal. Follow the instructions precisely.

- All cluster nodes are equal peers. The initial contact node has no permanent
  master role or special authority.
- The cluster's purpose is to enable computations whose data exceed the RAM of
  one machine. Its primary benefit is aggregate memory capacity, rather than
  lower latency than local computation or reliance on swapping.
- Shards of one table may reside on different nodes. Existing `scan*` operations
  and INSERT either claim or route through the cluster when the required shard
  is not local or RAM is not sufficient or pre-planned for other purposes.
  Placement is transparent to Scheme programs: `scan_order`, for example,
  returns the same result with its existing ordering and visibility semantics.
  Every scan variant retains its contract; a separate special-purpose API alone
  is not a complete cluster integration.
- Feature extensions, including new SQL frontend capabilities and planner
  operators, MUST remain cluster-capable and support distributed execution.
  Make sure that the cluster testsuite covers newly introduced critical cases.
  Prefer local execution when its data, authority and runtime state are local.
  Otherwise synchronize the required state and effects through the cluster
  protocol and delegate work as needed. Session state, transaction visibility,
  captured helpers, cancellation and side effects must preserve their native
  semantics across nodes. Use the already-implemented MOSI mechanisms or
  message passing for synchronization of shared objects.
  Tests for new capabilities must cover the distributed path as well as local
  execution; unsupported distributed semantics must be explicit, never hidden.
- Filters and MapReducers, including their captured bindings, are transmitted
  using the existing Scheme serializer and executed on the worker that reads
  the shard. Workers return scalar or grouped partial results for combination
  via message passing. Use the same coordination mechanics as the local goroutines:
  fire-and-forget delegation of the remote task, collect-and-count via a registered
  return channel.
  Distributed execution uses the existing planner and storage operators.
- Every shard access must be protected by `GetRead` or `GetWrite`. The optimistic
  already-authorized path neither locks nor blocks. These access boundaries
  determine whether the shard can be loaded locally for reading or mutation,
  primarily according to available RAM. When local RAM is insufficient, the
  acquisition raises an out-of-RAM panic; the caller catches it and decides
  whether the operation can be delegated to another node with capacity.
- When we acquire read or write on a shard that is held or owned by another
  peer, we must have a cost-based decision algorithm whether to claim that
  shard or to pass the insert or shard-scan command via /cluster message passing.
- Schema resources are identified by database name; shard resources by shard
  UUID. A rebuild creates a new shard with a different UUID. Writes are forwarded
  between the original shard and its rebuild successor while both remain live;
  existing operations retain valid shard identities and visibility throughout.
- Cluster nodes share the same persistent storage. Replication maintains a
  replica in separate storage. Both features use the same MOSI infrastructure
  and node protocol and can be combined on a node, including for different
  databases. Cluster peers trust one another; replication slaves are clients
  that must authenticate and whose access rights must be checked.
- The existing transaction system supplies atomicity and read visibility across
  nodes, including its distinct transaction publication modes. Remote execution
  preserves the scan's `maxRecId` bound to exclude later inserts and retains
  the required deletion and transaction-visibility checks. Mutations and later
  computations, including cached aggregates, observe the same coherence rules
  as local execution.
- Only the peer holding O tracks the other peers holding S for that shard.
  The S peers must remember the address of the owner.
  Nonlocal objects are represented by stubs, which may retain the URI of the
  O/M owner so commands can be forwarded to the peer. A peer need not track tables
  it has not loaded; it may discover owners by broadcast when loading them.
- The owner or modifier (MOSI) of a schema (database) must know the peer location
  of each of the table's shards. If he has an I(invalid) for a shard, it is known
  fact that no one has the shard and any peer can load it from the backend.

- Nodes may join and leave a running cluster. An orderly shutdown, including
  that of the initial node, releases all held read and write rights and hands
  over ownership responsibilities (Invalid preferred so other nodes can reload
  from storage when needed) so the remaining cluster continues running.
  Clients can reconnect to another node before shutdown, enabling uninterrupted
  migration.
- Failure of a worker holding no schema or shard ownership or permissions does
  not prevent the remaining cluster from operating. If a failed worker holds
  ownership or permissions needed for progress, the system waits for its return
  or an explicit administrative decision to kick it; a timeout never grants
  those rights to another node. A kicked node must discard its inconsistent
  state before rejoining, even if its process was still alive.

- Owner endpoints share `ClusterRequestHandler.HandleClusterRequest`; runtime
  closure values additionally implement `ClusterObject.Serialize`. Routing-only
  nodes, databases and coherence cells need no value serialization. Ownership
  policy stays with the native object, not a transport type switch.
- No cancellation check is added to shard item loops.
- Strong owner registrations retain physical generations while independent
  peer tokens exist. Imported values own GC-cleaned leases; duplicate imports of
  one token share one lifetime. GC only queues release, and a worker delivers it.
  A last release removes only that registration, never current partition rights.
  Disconnect does not revoke acquired handles or establish deletion proof.
  Captured session/cachemap views resolve before execution and provide local
  immutable reads; mutations use their owner endpoint. No extra shard read lock,
  row-level registration, counter or CAS may be added for these references.

## Parallel execution and publication

MemCP must execute independent work in parallel, using as few locks as possible.
The rules below describe required behavior, not a claim that every existing
implementation already satisfies it. A change must identify any remaining
violations; fixing a race by adding locks to an inner hot path is not sufficient.
Existing protection must not be removed before its replacement provides the
same ownership, visibility, lifetime, and durability guarantees.

TLDR: The goal is as few locking as possible
e.g. Table Statistics and other boilerplate structures must be atomically published via a CAS-pointer and must be fetched in O(1) by the planner; Don't update them at every insert, rather use the rebuild cycle to update statistics

### Immutable main storage and mutable deltas

- A shard's main column storages contain fixed, read-only values. Updates must
  not overwrite these values. Delta storage contains
  the insert list and deletion masks; transaction-local deletion/visibility
  overlays remain separate from the shared main storage.
- Delta writes must be thread-safe and lock-free:
  the inserts array must be CAS-swapped; readers hold maxRecId to avoid
  eternal loops in bulk-UPDATE commands (so they don't read the updated rows, too)
  the deletions mask is already covered by the CAS-based non-locking bitmap structure
  Reserve and initialize insert capacity to avoid one allocation per insert.
- delete masks can be overshadowed by the transaction's shard-local overlays
  to implement ACID visibility
  readers must respect the delete masks, indices must cover all items (even if
  it breaks uniqueness rules) and index consumers must filter out already-deleted
  items

### Readers and rebuilds must not wait for writers

- Readers must not be blocked by writers. They must be able to consume a valid
  published shard with the visibility required by their transaction while
  writes proceed. This requirement does not weaken SQL isolation or permit
  uncommitted values to become visible.
- Rebuilds must block neither reads nor writes. Compression, index construction,
  persistence I/O, and other shard-building work run on privately owned
  state, outside locks needed by active readers or writers. Publication must not
  turn this work into a stop-the-world phase.
- During a rebuild, writes must be forwarded to the original and new shards
  while both remain live. The handover protocol must cover inserts, updates, deletes, and
  commit/rollback visibility changes, including mutations concurrent with the
  capture of the rebuild input and final publication. Buffered forwarding during
  private construction is valid only if writes continue and the successor cannot
  be consumed until all required mutations have been incorporated. No mutation may
  be lost, applied twice logically, or resurrect a deleted row.
- Reads remain valid throughout the handover. Publish a complete successor;
  never expose partially initialized columns or mix shards to construct a
  row. An aborted rebuild leaves the original shard usable. Retaining old
  shards must also preserve the existing WAL and durable schema-publication
  contract; reclamation is not permission to delete committed data.
- A bounded mutex critical section is not automatically nonblocking. If readers
  or writers wait for a rebuild's catch-up/publication lock, that remains a gap
  against this contract and must be addressed explicitly.

### No synchronization writes in inner read hot paths

- `sync.RWMutex.RLock` and `RUnlock` write shared memory and cause cache-line
  traffic. They are not suitable for inner scan, filter, projection, comparison,
  or JIT-fused loops; pending writers can also block new readers.
  So synchronization must be at shard-level or higher.
- Reading must be lock-free and ideally perform no non-local RAM writes at all.
  Invocation-local cursor/accumulator updates are allowed. Atomic writes to
  shared reader counters or usage metadata are still non-local RAM writes;
  replacing a read lock with such a write does not eliminate that traffic.
- Move synchronization and metadata or statistics computation to publication, rebuild,
  operator setup, or batch boundaries. Row loops use invocation-owned readers
  and immutable main payloads. Shared usage counters, reference counters,
  feedback CAS operations, and cache admission must likewise stay out of those
  loops. Delta and transaction-overlay access still needs a valid concurrency
  protocol; main immutability cannot justify racing delta writers.
- Do not run query callbacks, compilation, I/O, or retained-memory traversals
  while holding locks needed by other operations. Define one lock order for
  remaining mutation/publication paths, avoid lock upgrades, and release locks
  and ownership rights on cancellation, errors, and panics.

### Bounded Work and Resource Lifetimes

Parallel work must respect node-local concurrency budgets, and streaming buffers must remain bounded. Every background task and retained resource must have an explicit owner and a completion or release protocol covering success, early termination, cancellation, errors and panics. Invocation-owned work must finish or be cancelled and joined before its state is released.

### Planner schema and statistics reads

- Schema and statistics queries must be lock-free and O(1) per requested
  metadata item, independent of row count and shard count. Enumerating or
  returning `k` metadata items naturally costs O(k); the implementation must not
  traverse the underlying data to obtain them.
- Publish consolidated metadata atomically for lock-free reads. Build
  schema descriptions and bounded statistical summaries during DDL,
  initialization, rebuild, or maintenance; do not recompute them when the
  planner or a plan-cache guard asks for them. A missing estimate is unknown,
  not permission to load columns, acquire locks, sample rows, or build indexes
  in a metadata getter.
- A reader must see compatible schema semantics and statistics, rather than
  combine incompatible publication revisions. Statistics-dependent choices
  retain the executable tipping-point guards described below. Explicitly
  costed sampling/preparation is a separate operation, never a hidden fallback
  inside a schema/statistics read or guard.

## Persistence and permanent format compatibility

### Compression formats and versioned headers

- Every persisted storage/compression format remains readable forever by later
  MemCP versions. Removing a writer or replacing a compression algorithm does
  not authorize removing its reader, reinterpreting old bytes, or requiring an
  export/reimport to open existing data.
- Format identity is stored in the file header, not inferred from the running
  MemCP release. The storage magic byte identifies the format family; a
  per-format version field selects its binary layout where that field exists.
  Magic bytes and published layout/encoding identifiers are permanent
  assignments: never reuse an old identifier for a different interpretation.
- A layout change needs a new version and a reader for that version. Writers
  emit their current documented layout; dispatch retains every historical
  reader, including `deserializeXxxV*` helpers and nested component formats.
  Never repurpose an old reader to decode a new layout. Optimization of an
  in-memory representation must not silently change the meaning of disk bytes.
- Legacy layouts without a version field retain their original framing. In
  particular, magic bytes 1, 2, 13 and 40 must not gain an inline version byte;
  changed layouts need a new magic byte and the original legacy reader remains.
  Apply the same rule to any other unversioned layout without a safely reserved
  version field. Existing padding or legacy sentinels may only be interpreted
  according to the documented compatibility rules for that format.
- Headers, byte order, integer widths, compression/encoding IDs, framing and
  legacy dispatch must be documented with the serializer/deserializer. Mixed
  old and new column formats must remain usable in one database; rebuilds may
  write current formats without making untouched old files unreadable.
- Unsupported required format versions and malformed data must fail explicitly,
  without overwriting the original data or substituting empty values. Optional
  statistics/cache hints may be discarded only when the public data and SQL
  semantics remain recoverable; this exception never applies to user data.

The `storages` registration and versioning comments in `storage/storage.go`
define the current dispatch assignments and legacy exceptions.

### Durability and cleanup ownership

- Preserve each engine's durability contract: `safe` uses fsynced WAL and crash
  recovery; `logged` uses WAL without fsync; `sloppy` persists rebuilt columns
  but has no WAL for subsequent deltas; `memory` has no restart durability.
  Performance work must not silently weaken the selected engine's guarantee.
- Rebuild, publication, log replay and retirement must preserve committed data
  through restart and failure according to that contract. An incomplete
  successor must not replace the last recoverable shard. Aborting a private
  file writer must not delete the published file.
- Eviction releases resident representations, never persistent user data.
  Memory-engine data and unflushed deltas must not be evicted in a way that loses
  them. File deletion belongs to explicit DROP or the documented explicit
  persisted-to-memory engine transition, not background cache cleanup.
- Lifecycle triggers must not delete unrelated data except through explicitly
  configured cascading policies. Preserve the engine-transition and cleanup
  contracts in `AGENTS.md`; format compatibility does not authorize destructive
  migration or weaken those contracts.

### Compatibility evidence

- A current-writer/current-reader round trip alone does not prove compatibility.
  Format changes need tests that read historical encodings and check values,
  NULLs and relevant boundary cases. Retain that coverage when writers change.
- Upgrade tests must open untouched data produced by predecessor binaries,
  validate public SQL results, and cover mutation and restart under the new
  binary. A successful startup alone is not sufficient. Extend the existing
  upgrade workload for new persisted optimization shapes.
- Keep the predecessor matrix and immutable `upgrade-release-*` tags described
  in `tests/storage/persistence/UPGRADE.md`; never move, reuse or remove a pin to
  hide incompatibility. Permanent read support is the contract even for formats
  not exercised by the current fixtures.

## Planner Pipeline

The planner pipeline is:

```text
parser AST
-> normalize_sql_syntax
-> decorrelate_logical_query / untangle_query
-> logical predicate placement and join_reorder / optimize
-> build_queryplan, scm-optimize and JIT
```

Each phase has a distinct job:

- parser: preserve SQL syntax and produce neutral AST/query terms
- `normalize_sql_syntax`: remove parser-specific spelling and syntactic sugar
  condensing syntax variants like IN vs. EXISTS SELECT to one form;
  normalization must be context-safe for SQL three-valued logic
- `decorrelate_logical_query` / `untangle_query`: build a decorrelated logical
  IR using Neumann domains, equivalence classes, and representatives
  with the goal that subqueries are no longer subscans but outer joins on a
  extended domain so they are visible to join_reorder
- logical predicate placement and `join_reorder` / optimize: move selections
  only across proven-safe logical boundaries, choose logical order, and record
  costing facts after decorrelation has exposed the complete join graph
- `build_queryplan`: choose physical scan sources/operators, bind conditions to
  concrete scans, and emit Scheme code

Do not move physical decisions into `untangle_query`. Do not leave logical
decorrelation work for `build_queryplan`.

SCM/Go optimizer interfaces operate on raw Scheme expressions only. They must
not know or hardcode a frontend's `(expression, type, collation)` record shape.
`tryfold` returns a constant value or raises for a non-foldable expression;
each frontend in `lib/` owns its type system and preserves or reconstructs its
own type and collation metadata. Failed folding must leave the original AST
available, and partially optimized execution code must not enter logical IR.

Each transformation has exactly one owning phase. A later phase may consume
facts established earlier, but it must not rediscover parser shapes or repeat a
semantic rewrite. In particular:

- syntax normalization must not choose RecSets, scans, keytables, or join order
- join reorder must run after dependent joins have been eliminated
- selection placement across query/group stages is logical; attaching a final
  predicate callback or index bound to a scan is physical
- physical lowering must consume canonical logical primitives instead of
  matching the original SQL spelling again

## Logical IR Shape

The logical planner is based on three combined operator shapes:

- `query-block`: select/join/filter/project/order/limit work unit
- `group-stage`: domain, aggregate, EXISTS/IN/scalar helper, HAVING, and window
  partition aggregate work unit
- `union-block`: set-operation work unit

`stage-output` is a logical relation descriptor produced by `group-stage` or
`union-block`. It is not a fourth algebra operator.

`parameter-rows` is a bounded logical VALUES relation for independent literal
or parameter cells. Its size comes from query syntax, never scanned data.
Cells remain request-bound through physical lowering, and UNION DISTINCT must
deduplicate their runtime values, preserving NULL versus zero. Substituting a
constant column still requires guards for every parameter used in that proof.
The descriptor does not authorize dropping domain keys, nullable partitions,
hidden columns, or ORDER/LIMIT barriers.

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

For Neumann decorrelation, `D` denotes the complete outer domain required by a
dependent subtree. An equivalence-class representative from the nullable side
of an outer join cannot replace `D`: missing matches would erase the very
domain rows for which `EXISTS` must produce `FALSE` or a scalar aggregate must
produce its empty-input result. Representative selection must preserve every
outer domain row before later selection placement or join reordering.

Positive membership in a WHERE truth context may use the semantic
`membership_truth` expression primitive. It references a logical group-stage
output and is not a physical RecSet or scan. Canonicalization may flatten
`IN (… UNION …)` to a n-ary OR of these primitives because SQL `UNKNOWN` and
`FALSE` both reject the row in that context. This rewrite must not cross `NOT`,
`CASE`, projection, or another value-producing context; those retain the full
match/RHS-NULL representation required for three-valued logic.

Rewrites must preserve every predicate and update every consumer of a replaced source, including hidden expressions. Terms that cannot move remain as residual predicates at a semantically valid boundary. Predicate placement must preserve multiplicity and outer-join null extension, including the distinction between ON and WHERE. IN and NOT IN retain SQL three-valued logic; presence-only rewrites require proof of equivalence in their enclosing context.

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

The plan cache may specialize for cost-model decisions or runtime values. Every specialization must guard the assumptions required for correct reuse. Value specialization must provide a concrete execution benefit; differing parameter values alone do not justify separate plans. Variant count and retained plan memory must remain bounded, with a reusable parameterized path for values not covered by retained specializations.

The queryplan cache may contain multiple lazily compiled plans for the same query
encoded by an if cascade. The condition must check if the tipping points of e.g.
table statistics are exceeded that would lead the planner to produce a different
operator selection. The value for each condition is the specialized query plan itself.
The last parameter of if (fallback value) is a call to the query planner itself to replan
the query.

The condition is one of the hot paths, especially for small query, hence we must keep the
condition as compact and as low-cost as possible. If a query is specialized to e.g. session parameters
these checks are mandatory. Only O(1) lookups are allowed (e.g. fetching table statistics),
no scans or such (except for some rare cases where some recset-probes or similar values
are dominant for all query plans, so they can be computed before the if and the guard
looks up their count. To keep the hot path guard evaluation fast, only the most dominant
tipping points of this planner session are to put into the guard.
They can be budgeted either by a constant number (not more than 5) or depending on the estimated
query runtime (only 1 planner guard for queries estimated at <1us if that runtime is exceeded).
Dominance in this context means: changing this planner decision has the highest difference in
cost estimation comparison. (e.g. the tipping point with 400% faster code is more dominant
than a decision with only 30% speedup compared to the second-best plan).

We must not add false assertions (like comparing the hash of a table statistic content
or asserting the equality of a specific table count) to the guard, instead it must be the
real cost model formula tipping point,
e.g. `constant1+constant2*table1.count < constant3+constant4*table2.count`,
so it must remodel the real planner decision for instance including cardinalities and
selectivity statistics if the planner relied on that.

Newer plans and their conditions must be pushed to the front of the if cascade.

## Relational Results Stay In The Storage Engine

You must not materialize or copy relational results into lists for sorting, limiting,
filtering or other operations like UNION variants. This bypasses our storage engine and hence
disables scalability on large table sizes. Always prefer a non-materialized direct-scan path
or if materialization is necessary, use reusable regular temporary tables with canonical names
which provide full index and partition support.
The only exception to use lists for intermediate relational materialization is when you gain
a speedup >50% e.g. on a smaller dataset and additionally add a guard to the
queryplanner cache as soon as the tipping point of that speedup is gone.
Example: WHERE ... IN (SELECT small item set FROM table2) may be materialized into a sorted
scm list to use a faster operator in favor of an exist-subscan until the
amount of X items in table2 is reached.

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

MemCP's logical decorrelation contract follows the model described in:

- `papers/Unnesting-Arbitrary-Queries.pdf`
- `papers/neumann-improving-unnesting-btw2025.pdf`

First attempt simple elimination of each dependent join: along paths of linear
operators, move selections into the join condition or maps above the join when
semantics permit. If no operator still accesses its outer bindings, convert it
into a regular join (NK15 Section 3.1; BTW2025 Section 3.2, Figure 3).

Otherwise use general top-down unnesting. Share the dependent join's domain and
outer-reference information, with a parent link for nested dependencies; keep
equality classes (`cclasses`) and representative substitutions (`repr`) local to
each query fragment. Separate join branches get separate rewrite states, merged
according to join semantics (BTW2025 Figures 4 and 6, Section 3.3).

Process nested dependent joins together in their parent context. Do not push
independently computed domains across dependent joins, creating Cartesian
products that later predicates discard (BTW2025 Section 2.3).

At a fragment with no remaining dependent accesses, either join its input with
D or substitute proven equivalent columns. Compare their costs: substitution
can lose selective domain pruning and increase intermediate work (NK15 Section
4; BTW2025 Section 3.2). General unnesting does not require materializing D.
These papers describe logical rewrites, not MemCP's persistent group-cache
implementation, and do not establish which SQL constructs MemCP supports.

## Domain D

For a dependent join with outer input L and inner input R, Domain D is the
DISTINCT projection of L onto `A(L) intersect F(R)`: the attributes L produces
that R reads as free variables. It contains the joint bindings actually present
in L, not a Cartesian product of distinct values from individual columns
(NK15 Section 3.2; BTW2025 Section 3.2).

Deduplicate D, not L or the subquery result. Joining the per-binding results back
to L must preserve outer multiplicity and the original join's unmatched-row
semantics. Domain matching uses null-safe equality. Original SQL comparisons
retain their own NULL semantics: substituting equal columns does not authorize
dropping a NULL-rejecting predicate such as `x = x` (NK15 Section 3.3).

In MemCP, semantically relevant external dependencies, including session reads,
are part of the complete Domain D. Carry their values explicitly through the
helper's domain/key context, including dependencies below grouped or derived
blocks. This is a binding requirement, extending the papers' free-variable
model; neither paper specifies persistent group caches or session-key persistence.

Physical group caches represent domain bindings (or their proven equivalent
representatives) and relevant session values as explicit logical key dimensions,
alongside any grouping, range or ordered coordinates needed by the helper.
They may be reused across queries and sessions when those keys distinguish
every semantic binding and source dependencies remain valid. Cache rows retain
values, never session or transaction objects. This is separate from the
query-local lifetime of physical RecSets and RecMaps.

Shared cache computation, publication and reuse must preserve source consistency and transaction visibility. Concurrent source changes must not allow stale results to be published or reused as current. Reusable cache entries retain logical keys and values, never request-local sessions, transactions, closures or physical row identities. Session-values (like SQL @userid oder @refrenced_time) can either be bound into the Domain (especially for small sets for domain keys, large spans in ranged domain dimensions) or you can compile the plan in a way that the innermost caches avoid using such external references at all.

The internal transaction session `__memcp_tx` identifies execution state, not
SQL input, and must not become a semantic domain key. All other session reads
that can change a helper result are external dependencies, including values
used below grouped or derived query blocks.

A scalar subquery returns NULL for zero rows, the selected value for one row, and an error for multiple rows. An explicit SQL row limit retains its own semantics; optimization must never silently introduce LIMIT 1. Rather carry cardinality and overflow semantics explicitly through logical rewrites. Correlated ORDER BY, OFFSET and LIMIT apply separately to each domain binding.

## Groups and HAVING

Grouping is a logical stage boundary. Aggregates and HAVING belong to the group-stage; consumers access its stage outputs.

An explicit nonempty GROUP BY produces no groups for empty input; a scalar
lookup of that missing result yields NULL. Grouping sets containing the empty
set preserve its aggregate row too (BTW2025 Section 3.3). Preserve unmatched
outer bindings according to the enclosing join's contract.

Physical: When using wider carriers for the group cache

## Derived Tables and Materialization

Ordinary `FROM (SELECT ...)` must be inlined early by `untangle_query`.

Inlining builds a renaming map, prunes unused visible and hidden slots, and
merges predicates after renaming. Ordinary derived-table materialization is not
a default strategy.

Materialize only when a real semantic or physical barrier exists:

- reusable group/aggregate cache
- conflicting window order requiring ORC
- shared CTE/DAG root
- physical cost model chooses a query-local scan source such as RecSet
- temp table as last-resort physical barrier

## UNION

Logical UNION stays as `union-block`.

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
- shard-parallelity and faster execution when work is independent

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

## Canonical Naming and Cache Reuse

Helper identities must be canonical. This means: Similar queries that share the same subformula or same grouping scheme or same aggregation formula or same subquery must result in the same shared helper table or column.

Derive physical helper names from:

- source identity (physical column name); name aliasses are irrelevant and must be normalized to physical (done in the hashing function - materialization-free)
- full normalized key set (for group table names)
- canonical filter/condition forms when they affect the cached or materialized result
- semantic facts that change the result

Never derive helper identity from visible aliases, wrapper-local aliases, or
projection titles.

The same keytable plus the same aggregate formula should produce the same
computed column. Prefer reusable full key domains over many narrow one-off
keytables when additional filters can be applied while reading stage output.

Multiple distinct aggregate formulas on the same keyset must end as columns
on the same keytables.
If a keyset equals a Primary key or unique key of a table, use that table as carrier instead.

## Tests and Review Gates

A branch touching query planning must preserve the architecture described in INVARIANTS
and must not introduce performance regressions in other queries.
For this purpose, every major performance improvement (>30%) in the planner
must introduce a new performance test if not already covered, so a
performance regression in future PRs will be noticed.

If testing temporarily, the test dataset should be not too small,
otherwise it leads to illegal optimizations, e.g. copy all items to a
scm list and continue computing on that list which blows up on bigger
datasets and is not scalable.

Do not weaken tests, mark critical tests noncritical, or replace performance
guards with shape-only assertions.

### SQL compile-time type and collation ownership

- SQL datatypes must NEVER be inherited from the SCM runtime type. The SQL
  frontend determines types from SQL syntax, declared column metadata, and
  expression contracts, independently of the internal value representation.
- The expression compiler returns a triple `(expr, type, collation)` for each
  expression. The final compiled code must contain only `expr`. `type` and
  `collation` are used for SQL compile-time decisions such as selecting
  collations for storage operators, compiling casts, or reporting type errors;
  they must not survive as runtime type wrappers or checks in storage operators.

### Storage is opaque to Scheme cache and plan policy

Table handles identify logical relations. Shard identities, topology changes,
cache eviction events, readiness probes and usage touches are not planner
inputs. Physical topology is visible through explicit `show` diagnostics only.
`shardcolumn` / `partitiontable` remain declarative range-partitioning hints.
Ordered plan selection must not depend on the current physical partition layout.
The planner must construct the ideal queryplan so that column-partition weighting
can adapt over multiple execution. The same holds for indices: The planner must
assume that every possible index exists on every possible column combination
exists. The autoindex mechanism of memcp will create those indexes as soon as
the search cost exceed the index buildup cost.

