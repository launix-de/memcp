# MemCP Beta Release 0.9: One Database for Transactions, Search, and Analytics

Today we are releasing MemCP 0.9 as a public beta.

MemCP is a persistent main-memory database designed to run existing SQL applications without giving up the advantages of a compact, column-oriented engine. Applications can connect through the MySQL protocol, send MySQL- or PostgreSQL-style SQL over HTTP, use the built-in REST router, or query RDF data with SPARQL. Data remains durable on disk while frequently used columns, indexes, and intermediate results stay close to the CPU.

The 0.9 release marks a change in what MemCP is ready to do. The first public release demonstrated the storage model and broad SQL support. The engineering work summarized in our [August update](https://launix.de/launix/memcp-engineering-update-mar-9-aug-26-2026/) showed that the same architecture could handle a demanding real application. MemCP 0.9 brings those pieces together as a database people can install, connect to existing software, observe, benchmark, and begin validating for real deployments.

This is still a beta, not a claim that every SQL edge case or operational environment is finished. It is, however, the point at which MemCP has become much more than a fast column scan: it is a complete database system with transactions, automatic physical optimization, crash recovery, memory management, migration tools, protocol compatibility, and an administration dashboard.

## What MemCP 0.9 Can Do

### Run Existing MySQL Applications

MemCP exposes a MySQL-compatible TCP server and Unix socket. Existing connectors, PDO applications, command-line clients, prepared statements, authentication flows, and administration tools can talk to it without a MemCP-specific driver.

The supported SQL surface now covers the workload expected from a modern business application:

- `SELECT`, `INSERT`, `UPDATE`, `DELETE`, upserts, and `INSERT ... SELECT`
- inner, outer, cross, and multi-table joins
- scalar, `IN`, `EXISTS`, `NOT EXISTS`, and correlated subqueries
- `GROUP BY`, `HAVING`, distinct results, common aggregates, and `GROUP_CONCAT`
- `UNION` and `UNION ALL`, including ordered and correlated forms
- derived tables, logical views, complex expressions, and JSON functions
- window functions such as `ROW_NUMBER`, `RANK`, `LEAD`, and `LAG`
- triggers with `OLD` and `NEW` values, including query-backed trigger bodies
- prepared binary values, session variables, collations, date/time functions, and timezone-aware conversion
- schema inspection through the common `SHOW` and `INFORMATION_SCHEMA` interfaces

Compatibility work in 0.9 was driven by unmodified application queries, database checkers, plugins, notifications, generated read models, and dump/restore tools. This matters more than accepting isolated syntax: the parser, planner, transaction layer, and result protocol now cooperate on the shapes real software actually produces.

MemCP also provides HTTP endpoints for MySQL-dialect SQL, PostgreSQL-dialect SQL, Scheme, and SPARQL. Applications can register custom HTTP routes in Scheme and execute directly against the storage engine without another application-server-to-database network hop.

### Keep Data Durable While Serving It from Memory

MemCP is not an ephemeral cache. Its default `safe` engine persists data with a write-ahead log and recovery path designed for power-loss safety. The `logged`, `sloppy`, `memory`, and `cache` engines offer explicit alternatives for workloads with different durability and reconstruction requirements.

Tables are divided into independently managed shards. Each shard stores columns separately and selects a compact representation according to the data: bit-packed integers, decimal encodings, dictionaries, prefixes, sparse values, sequences, constants, or compressed string storage. Queries read only the columns they need. Cold persistent data can leave RAM without being deleted from disk and is loaded again when required.

Release 0.9 substantially strengthens this operational foundation:

- shard rebuilds and repartitioning publish new generations atomically
- concurrent inserts, updates, and deletes survive generation switches
- WAL replay restores committed changes after a crash
- persistent blob references and large WAL records survive restart
- orphaned blob and shard files can be reclaimed without connecting cleanup to normal cache eviction
- memory accounting separates owned structures and gives eviction an accurate budget
- cold shards, indexes, computed columns, and reconstructible caches can be released under pressure
- background maintenance avoids deleting persistent data and keeps the previous valid generation if publication fails

The result is a hot working-set database rather than a requirement that every byte remain permanently resident. MemCP can use available memory aggressively while retaining an explicit durability contract.

### Optimize Queries Without Manual Index Administration

MemCP builds and maintains physical access paths from the workload. Users do not have to predict every useful compound index in advance or run a manual `ANALYZE` cycle before the optimizer can improve.

The query compiler separates SQL meaning from physical execution:

1. The parser builds the SQL syntax tree.
2. Correlated subqueries are decorrelated into a combined logical graph.
3. Logical optimization and join reordering consider the complete query, including formerly dependent joins.
4. Physical lowering chooses scans, ordered scans, probes, cached groups, computed columns, and row-ID sets using a calibrated cost model.
5. The resulting Scheme program is optimized and cached as an executable query-plan callable.

This separation is one of the central achievements of 0.9. A permission check expressed as nested `CASE`, `COALESCE`, `EXISTS`, and scalar subqueries is no longer forced down one hard-coded execution path. The compiler can choose independently at different nodes of the plan: probe a unique key, scan an index range, build a reusable group, project a candidate set through a join, or evaluate a residual predicate in batches.

Physical decisions use table cardinality, selectivity, distinct counts, ordering, limits, projected work, and runtime guards. When better statistics become available or data growth crosses a decision boundary, a cached plan can be recompiled instead of remaining tied to an obsolete assumption. A calibration suite measures the real storage operators and generates the constants used by the cost model; performance-sensitive pull requests are also compared against master across a checked-in workload pool.

### Search, Filter, and Paginate Large Tables

MemCP 0.9 combines several mechanisms for interactive search over large business datasets:

**Adaptive automatic indexes** recognize equality, range, ordering, and indexable matcher boundaries. Non-prefix `LIKE '%term%'` predicates can use a shard-local bigram candidate index. The original SQL predicate remains the correctness authority, so an approximate candidate set can skip most rows without changing the result.

**RecSets** represent query-local sets of physical row IDs. A RecSet chooses ranges for clustered matches, sorted ID lists for sparse matches, or bitmaps for dense scattered matches. Sets can be intersected, united, negated, counted, scanned, and projected through join keys without copying complete rows.

**Ordered scans with braking** exploit an index that already supplies the requested order. For `ORDER BY ... LIMIT`, the engine can stop when later candidates cannot enter the result. When a selective RecSet and a different ordering index compete, the scan operator can choose between walking the ordered index with membership checks and sorting the sparse candidates by their inverse index positions.

**Batch acceptance** helps when a cheap condition can produce candidates but an expensive permission predicate must still be checked. The engine pulls a bounded candidate batch, evaluates the remaining condition, and expands the batch only if it has not filled the requested page.

These are general operators rather than a special full-text subsystem. The same machinery can combine text search, tenant or location restrictions, access-control subqueries, an ordering requirement, and a page limit. Low- and high-selectivity searches can therefore choose different execution paths from the same SQL shape.

### Reuse Computation Safely

Main-memory execution is most valuable when repeated work is recognized across rows and queries.

MemCP can create computed columns for recurring expressions and scalar lookups. A lookup such as fetching a value through a stable foreign key can become a lazily computed column on the referencing table. Canonical naming allows compatible queries to reuse it, while dependency tracking invalidates the affected row when a bound key or source value changes.

For aggregation, group caches preserve reusable partial results and update them as base data changes. Permission-aware counts can sometimes be evaluated once per low-cardinality group rather than once per fact row. Query-local scalar and membership carriers avoid rebuilding the same subquery result for every projection or filter occurrence.

Reuse is always scoped by its dependencies. Session-bound permissions do not leak between users, transaction-visible values are not published as global facts, and residual correlated expressions remain inside the row scope that supplies their keys.

### Serve Small OLTP Requests and Large Scans in One Engine

Earlier MemCP versions were most obviously strong when a query touched enough data for compression, columnar access, and parallelism to dominate. Version 0.9 also reduces the fixed cost of tiny requests.

On a realistic WordPress-style fixture with 8,214 posts and 75,000 metadata rows, serial requests through the MySQL protocol measured:

| Query | MemCP 0.9 | MariaDB 10.11 | Result |
| --- | ---: | ---: | ---: |
| Primary-key lookup | 0.06748 ms | 0.06864 ms | MemCP 1.7% faster |
| `ORDER BY post_title LIMIT 5` | 0.15829 ms | 2.44767 ms | MemCP 15.46x faster |
| Filtered posts/metadata join with `ORDER BY ... LIMIT 20` | 0.24977 ms | 0.26430 ms | MemCP 5.8% faster |

The comparison used the same data distribution, 500 warm-up requests, and seven measured blocks of 5,000 serial requests per query. These numbers are deliberately end-to-end protocol measurements rather than isolated storage calls.

The first real-world beta integration provides the other side of the picture: approximately 855,000 central records, generated SQL, nested permission rules, counts, pagination, direct-column search, and file-content search. Interactions that once took minutes are now generally interactive after warm-up. On representative search workloads, MemCP has measured up to 30x faster than PostgreSQL. This is not a promise that every query will be 30x faster; it demonstrates that MemCP's automatic indexes, candidate sets, and compiled permission paths can produce a decisive gain on an application that was not designed around MemCP.

The important release result is not one winning number. It is that point lookups, ordered pages, joins, broad aggregations, and complex searches can now compete inside one physical planner instead of requiring separate transactional and analytical databases.

### Import, Package, Observe, and Operate

MemCP 0.9 is distributed as more than a source checkout:

- Debian and RPM packages with systemd integration
- release binaries and checksums
- a multi-architecture container image
- explicit initialization and secure first-start password handling
- configuration files suitable for service deployment
- MySQL dump round-trip support
- PostgreSQL dump import, including `COPY`, schemas, sequences, and common dump DDL
- CSV and JSON bulk loading
- local filesystem, S3-compatible, and Ceph/RADOS persistence backends

The web dashboard shows databases, tables, columns, shards, compression, memory ownership, active connections, errors, and running queries. Operators can inspect settings and cancel stuck work. `SHOW FULL PROCESSLIST` reports the active SQL text, while persistent error-query logging keeps failure context available after the request ends.

Planning is observable as well. `EXPLAIN`, `EXPLAIN IR`, `EXPLAIN REORDER`, and `EXPLAIN PHYSICAL` expose the generated Scheme program, logical graph, join-order decision, and physical alternatives. Trace output is complete by default; administrators may configure an explicit visible truncation limit when logs must be bounded.

## What Changed Since the August Engineering Update

The August report concentrated on the long path from multi-minute correlated queries to interactive execution. The final work for 0.9 broadened and hardened the whole system around that result.

### The Physical Planner Became a Shared Decision Layer

Ordered join scans, projected candidate carriers, adaptive RecSet boundaries, batch-local predicates, range braking, and low-cardinality grouped joins now enter the same costed search space. Compiler invariants explicitly prohibit physical artifacts from leaking into the logical phase. This makes new operators available to arbitrary compatible SQL shapes instead of adding another query-specific lowering branch.

The cost model is generated from measured primitives and exercised by a calibration suite. CI compares branches with master across point, ordered, joined, grouped, correlated, and long-running workloads, rejecting a local optimization when it causes an unacceptable regression elsewhere.

### The Execution Path Became Cheaper

Scan analysis and setup were reorganized around reusable scratch storage. Common filter/map/reduce callbacks avoid per-row allocations. Planner list pipelines, range consumers, unique collectors, and grouped reductions fuse rather than construct intermediate lists. MySQL result rows stream directly into protocol packets instead of first becoming a second row representation.

Query execution context is now explicit. Removing goroutine-local transaction state made cancellation, query reporting, authentication scans, nested execution, and future compilation easier to reason about. Query plans are cached as explicit callables that can use the interpreter today and the experimental JIT when that backend is enabled.

### Compatibility and Failure Handling Were Tested as Workflows

The suite now includes application-shaped notifications, schema checks, generated permission trees, page selection, deletion, ID-list workflows, dump recovery, concurrent rebuilds, cancellation, package lifecycle, and process reporting. These tests do more than check parser acceptance: they execute the sequence in which an application relies on the feature.

The test infrastructure itself gained SQL time-limit guards, plan-shape assertions, master-versus-branch performance checks, crash/restart fixtures, and packaging validation. The goal is to turn failures found during beta use into permanent anonymous regressions.

## Why 0.9 Is a Beta

MemCP 0.9 is suitable for serious evaluation and controlled beta deployments, but the version number is intentional.

- SQL is a large language. Complex application suites should still be tested against their exact generated statements before cutover.
- The first production-style deployment is in beta validation; sustained concurrency, unusual session behavior, and long-term recovery remain active test areas.
- Cost estimates continue to improve as the calibration workload expands across data sizes and selectivities.
- PostgreSQL compatibility is currently a SQL dialect and import path over HTTP, not a drop-in PostgreSQL wire server.
- Foreign keys are accepted as metadata but are not generally enforced as relational constraints.
- The amd64 JIT remains experimental and requires a patched Go toolchain. Normal builds use the fully supported interpreter path.
- Remote object-storage configurations need the same operational testing and backup discipline as any other database deployment.

For important data, use the default `safe` engine, keep tested backups, validate restore procedures, and compare application results during migration. Beta means we want real workloads and feedback; it does not mean weakening the durability contract.

## A Database That Learns the Workload

MemCP's defining idea is now visible across the complete system.

Columns choose compact encodings from their values. Shards rebuild and repartition as data changes. Indexes emerge from query boundaries. Computed columns and grouped results retain repeated work. RecSets change representation with density. Ordered scans adapt between index-driven and candidate-driven traversal. Cached plans carry guards for the assumptions that made them cheap.

Applications continue to express what they need in SQL. MemCP decides how to store, restrict, order, reuse, and execute that work.

Version 0.9 is the first beta release where all of those layers operate together: a persistent main-memory storage engine, a general decorrelating query compiler, adaptive physical operators, familiar database interfaces, operational tooling, and deployment artifacts. The next phase is no longer proving that the architecture can execute a real application. It is running more of them, closing the remaining compatibility gaps, and turning beta evidence into a stable 1.0 contract.

## Get Started

Project and source code: https://github.com/launix-de/memcp

Previous report: https://launix.de/launix/memcp-engineering-update-mar-9-aug-26-2026/

MemCP 0.9 is open source under the GNU General Public License v3 or later. We welcome application compatibility reports, reproducible query shapes, performance comparisons, and operational feedback from beta users.
