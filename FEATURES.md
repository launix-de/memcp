<!-- Copyright (C) 2026 Carl-Philip Hänsch; GPL-3.0-or-later -->

# MemCP feature catalog

This catalog describes the capabilities present in the repository, from SQL
and application interfaces to physical operators and native compilation. It
was reviewed against development commit `e36686e00` in September 2026. A release
can contain a different subset; follow the source and test links when checking
a particular version. Update this catalog when adding or changing a capability.

MemCP is **Beta**. “Supported” below means implemented functionality with the
scope indicated, not complete compatibility with another database. The JIT is
**experimental**. Tests demonstrate specific cases, not every combination of
features. For installation and examples, start with the [README](README.md).

## Contents

- [Language frontends and application interfaces](#language-frontends-and-application-interfaces)
- [SQL support](#sql-support)
- [Query planning and execution](#query-planning-and-execution)
- [Storage backends and durability](#storage-backends-and-durability)
- [Internal storage representation](#internal-storage-representation)
- [Column storage types and compression](#column-storage-types-and-compression)
- [Internal storage operators](#internal-storage-operators)
- [Scheme runtime and JIT](#scheme-runtime-and-jit)
- [Operations, security, and data movement](#operations-security-and-data-movement)
- [Packaging and distribution](#packaging-and-distribution)
- [Validation and development tools](#validation-and-development-tools)

## Language frontends and application interfaces

| Interface | Capabilities and scope | Implementation / examples |
|---|---|---|
| MySQL-style SQL | SQL parsing, execution and metadata compatibility for application queries; available over HTTP and the MySQL protocol. | [Parser](lib/sql-parser.scm), [compatibility tests](tests/sql/compatibility/) |
| PostgreSQL-style SQL | Separate parser and `/psql/<database>` endpoint, with PostgreSQL syntax and expression adaptations. This is not a PostgreSQL wire-protocol server. | [Parser](lib/psql-parser.scm), [parser tests](tests/sql/compatibility/psql-parser.yaml) |
| MySQL wire protocol | TCP/Unix-socket access for existing clients, authentication, session state and prepared/binary statement support. Protocol compatibility does not imply complete MySQL SQL or administration compatibility. | [Server](scm/mysql.go), [binary statement tests](tests/sql/compatibility/mysql-prepared-binary.yaml) |
| SQL over HTTP | `/sql/<database>` and `/psql/<database>` query execution with result serialization and session handling. | [SQL handlers](lib/sql.scm), [HTTP runtime](scm/network.go) |
| Scheme | Embedded language for query compilation, storage calls and application handlers; REPL, module import and `/scm` evaluation. | [Runtime](scm/scm.go), [bootstrap](lib/main.scm) |
| RDF / SPARQL | RDF query/update processing, Turtle loading, graph patterns, OPTIONAL, UNION, filters, named graphs, aggregates and property-path cases. Consult individual tests for coverage and unsupported forms. | [RDF parser](lib/rdf-parser.scm), [RDF engine](lib/rdf.scm), [tests](tests/rdf/) |
| Embedded PHP | In-process application hosting through FrankenPHP, document-root mounts, front controllers and a `memcp:` PDO driver. Requires a PHP-enabled build and external PHP ZTS/embed dependencies. | [PHP guide](README-PHP.md), [bridge](phpbridge/) |
| Web applications | Scheme HTTP handlers, static serving, reverse-proxy handlers and WebSockets; handlers can share the database HTTP listener. | [Network runtime](scm/network.go), [application bootstrap](lib/main.scm) |

## SQL support

The MySQL and PostgreSQL parsers have different coverage. This section groups
implemented SQL functionality; it is not a claim that every spelling works in
both dialects. See [SQL tests](tests/sql/) and [planner tests](tests/planner/)
for executable examples and deliberately rejected cases.

| Area | Implemented functionality |
|---|---|
| Selection and projection | Expressions, aliases, qualified names, `WHERE`, `DISTINCT`, ordering, `LIMIT` and `OFFSET`. |
| Joins | Inner, left, right and cross joins, join predicates and multi-table query planning. |
| Subqueries | Scalar subqueries, derived tables, `EXISTS` / `NOT EXISTS`, `IN` / `NOT IN`, and correlated forms that the logical planner can decorrelate. Unsupported logical shapes must fail explicitly rather than silently use a correlated per-row fallback. |
| Set operations | `UNION` and `UNION ALL`, including duplicate handling and ordered/limited result cases. |
| Aggregation | `GROUP BY`, `HAVING`, `COUNT`, `SUM`, `AVG`, `MIN`, `MAX`, `GROUP_CONCAT`, grouped expressions and empty-input semantics. |
| Window functions | Ranking (`ROW_NUMBER`, `RANK`, `DENSE_RANK`), offsets (`LAG`, `LEAD`), value functions (`FIRST_VALUE`, `LAST_VALUE`) and window aggregate cases, with partition/order handling. Supported frames and grouped/window combinations are defined by the tests, not full SQL-standard window coverage. |
| Expressions | SQL NULL/three-valued logic, `CASE`, `COALESCE`, `NULLIF`, `IF`, arithmetic/bitwise operators, comparisons, `BETWEEN`, literal membership and type conversions. |
| Strings and search | String manipulation, `LIKE`/`ILIKE` dialect cases, regular expressions, collations and `MATCH … AGAINST` compatibility. Accepted full-text syntax does not promise MySQL-identical index or relevance semantics. |
| Numbers, dates and JSON | Numeric functions, date/time arithmetic and formatting, timezone handling, JSON extraction/construction and PostgreSQL JSON expression cases. See the [builtin definitions](lib/sql-builtins.scm) and [expression tests](tests/sql/expressions/). |
| INSERT | Single/multiple rows, `INSERT … SELECT`, MySQL `INSERT … SET`, ignore/upsert forms and dialect-specific conflict handling. |
| UPDATE and DELETE | Filtered mutations, qualified targets, self-references and supported multi-table/subquery forms. |
| Schema operations | Database/table creation and deletion, column changes, table renaming, truncation, indexes, views and per-table engine selection. |
| Column semantics | Numeric, text, date/time and binary types; defaults, default expressions, auto-increment, NULL restrictions and collations. Type spellings do not imply identical storage/range/coercion behavior to another database. |
| Keys and foreign keys | Primary/unique keys, foreign-key validation and tested RESTRICT, CASCADE and SET NULL actions. |
| Triggers and computed values | Persistent SQL triggers, BEFORE/AFTER mutation cases, trigger queries, computed columns and dependency-driven invalidation. |
| Metadata | `SHOW` commands and selected `INFORMATION_SCHEMA` relations for applications, clients and import tools. |
| Session and administration | Variables, prepared statements, users/grants, process inspection, query cancellation, table locks and dump/session compatibility statements. Some compatibility statements are accepted without reproducing another server's full semantics. |

Examples and boundaries: [joins](tests/planner/joins/),
[subqueries](tests/planner/subqueries/), [aggregates](tests/planner/aggregates/),
[windows and ordering](tests/planner/order-window/), [DML](tests/sql/dml/),
[DDL](tests/sql/ddl/), [triggers](tests/sql/triggers/),
[security](tests/sql/security/), [SQL types](lib/sql-types.scm).

### Transactions and concurrency

- `BEGIN` / `START TRANSACTION`, `COMMIT` and `ROLLBACK` provide the
  cursor-stability transaction path. Statements can run in implicit transactions.
- `START ACID TRANSACTION` selects the snapshot-isolation path with optimistic
  commit validation. This is a distinct mode, not a claim that ordinary `BEGIN`
  has serializable isolation.
- Transactional mutations, rollback, multi-shard commit/recovery and trigger
  effects have dedicated tests. Conflicts and statement failures are explicit
  error paths.
- `LOCK TABLES`, process/session tracking and query cancellation support
  application coordination and operations.
- Shards permit parallel scans; mutation, rebuild and repartition paths have
  concurrency tests. Supported concurrency does not mean all operations are
  lock-free or immune to transaction conflicts.

Sources: [transaction implementation](storage/transaction.go),
[SQL transaction syntax](lib/sql-parser.scm),
[concurrency tests](tests/execution/concurrency/),
[persistence tests](tests/storage/persistence/).

## Query planning and execution

| Capability | Purpose |
|---|---|
| Logical normalization and decorrelation | Turn supported correlated subqueries into explicit relational stages before physical execution planning. |
| Combined logical operators | `query-block`, `group-stage` and `union-block` keep related filtering, grouping and output requirements together. |
| Join reordering | Choose join order using dependencies, cardinality estimates and cost, rather than SQL text order alone. |
| Cost-based physical alternatives | Compare scan, ordered-access, membership, mapped-row and aggregation strategies. |
| Adaptive access paths | Build/reuse useful indexes for equality, ranges, ordering and applicable text predicates. |
| Filter statistics | Combine planner statistics, bounded selectivity estimates and feedback from completed scans. |
| Fused execution | Combine filtering, projection and reduction; read referenced columns and avoid unnecessary row materialization. |
| Ordered limits | Use ordered access, bounded candidate sets and early stopping where operator semantics permit it. |
| Group caches | Reuse aggregate state and maintain/invalidate it as source data changes; range-group caches reuse partial states across overlapping ranges. |
| Ordered computed columns | Reuse order-dependent computed values, including partition-aware invalidation paths. |
| Query-plan templates | Separate reusable query structure from runtime parameters; cache variants and invalidate/replan when relevant dependencies or statistics change. |
| Explain facilities | `EXPLAIN`, `EXPLAIN IR`, `EXPLAIN REORDER` and `EXPLAIN PHYSICAL` expose generated code and planning stages; `EXPLAIN COMPILE` reports compilation work. |

The [planner invariants](INVARIANTS.md) define the phase boundaries.
Implementation: [logical model](lib/queryplan-logical.scm),
[optimizer](lib/queryplan-optimize.scm),
[physical plan](lib/queryplan-physical-plan.scm),
[scan lowering](lib/queryplan-physical-scan.scm),
[SQL plan caching](lib/sql.scm). The [cost generator](tools/costgen/) measures
alternative operators to calibrate cost constants.

## Storage backends and durability

A **backend** selects where database objects live. An **engine mode** selects
the persistence/reconstruction contract of a table. They are different choices.

| Backend | Capability | Source |
|---|---|---|
| Local filesystem | Column/blob files, schema persistence and write-ahead logs. | [Filesystem backend](storage/persistence-files.go) |
| S3-compatible object storage | Database storage through an S3 endpoint, including AWS-style configuration and custom endpoints such as MinIO. | [S3 backend](storage/persistence-s3.go), [integration tests](storage/persistence_s3_integration_test.go) |
| Ceph RADOS | Direct RADOS storage; optional build requiring `librados` and the `ceph` build tag. | [Ceph backend](storage/persistence-ceph.go) |

| Table engine | Durability and eviction |
|---|---|
| `safe` (default) | WAL persistence at statement/transaction boundaries; committed writes are intended to survive process crashes and power loss when the backend honors its durability contract. Persistent data can be evicted from RAM. |
| `logged` | WAL without the local fsync guarantee; process-crash recovery, but recent writes can be lost after power failure. |
| `sloppy` | Flash-friendly persistence: batches changes into compressed column files instead of writing a WAL for every mutation. The background rebuild saves changes on a 15-minute schedule. Deltas since the last completed rebuild are lost after an unclean shutdown. |
| `memory` | Rows exist only in RAM, are not evictable and disappear on restart; schema and an optional reconstruction callback persist. |
| `cache` | Reconstructible RAM data that may be cleared under memory pressure; supports an initializer and persistent schema. |

Changing a persisted table to `memory` removes its persistent files and WAL.
Remote-backend guarantees depend on the storage service and its failure modes;
a local fsync description must not be read as an identical remote implementation.
See [persistence interfaces](storage/persistence.go),
[engine guidance](README.md#storage-engines) and
[reliability drills](tools/reliability-drill.md).

## Internal storage representation

- **Main/delta layout:** compressed stable columns plus recent mutations;
  rebuild folds changes into a new column generation.
- **Adaptive column encodings:** choose representations from the data, not
  the SQL type name alone; see [column storage types](#column-storage-types-and-compression).
- **Batch readers:** storage readers expose batches for scans and generated
  native consumers; compressed data need not become full row objects first.
- **Adaptive shards and indexes:** partitioning, index construction and
  main/delta access support changing data and workload shapes.
- **Computed storage:** temporary columns, group tables and ordered computed
  values cache derived work with dependency tracking.
- **Memory management:** budgets, eviction, compressed row sets, decoded-string
  caches and accounting distinguish retained data, reclaimable representations
  and process RSS. Eviction of persistent representations does not delete the
  persistent data.
- **Format compatibility:** versioned serialization and legacy readers support
  reopening older persisted formats; release-upgrade tests exercise this path.

Sources: [storage formats and registration](storage/storage.go),
[storage implementation](storage/), [format tests](tests/storage/formats/),
[index tests](tests/storage/indexes/),
[upgrade compatibility](tests/storage/persistence/UPGRADE.md).

## Column storage types and compression

Compression depends on value range, repetition, NULL density, string alphabet
and row order. The following are **encoding-derived payload estimates**, not
benchmark measurements or whole-database size promises. A ratio of **8×** means
one eighth of the reference size (87.5% saved). Numeric examples compare with
8-byte integers/floats; string examples compare with the original text bytes.
Generic-value examples use 16-byte `Scmer` slots on 64-bit systems, excluding
separately allocated objects. Headers, allocation rounding, indexes, mutable
deltas and decoded caches add space; RAM, serialized size and process RSS are
different quantities.

### Numeric, constant and sparse columns

| Type | Suitable data and representation | Attainable compression / size model |
| --- | --- | --- |
| [`StorageConst`](storage/storage-const.go) | Every row has the same value, including an all-NULL column. Stores one value and the row count. | No per-row value payload. For `n` repeated fixed-size values, the value-payload reduction is **n×**; total storage is one value plus fixed metadata. Small columns therefore achieve less than this payload ratio. |
| [`StorageInt`](storage/storage-int.go) | Integers with a narrow minimum-to-maximum range, even when the absolute values are large. Stores an offset and bit-packed differences; NULL reserves another code. | **64/b×**, where `b` is the required bit width, at least 1. Widths of 1, 3, 5, 8, 16 and 32 bits give **64×, 21.3×, 12.8×, 8×, 4× and 2×** respectively. A full-width 64-bit range gives **1×** before overhead. Outliers or an extra NULL code can increase the width for the whole column. |
| [`StorageSeq`](storage/storage-seq.go) | Long arithmetic sequences, such as consecutive IDs, regularly spaced timestamps, constant runs or NULL runs. Each segment stores its starting row, starting value and stride in three bit-packed integer columns. | Space grows with segment count `r`, rather than row count `n`. Counting three full 8-byte fields per segment gives **n/(3r)×** before metadata; 100 rows per segment give **33.3×**, with further savings possible from bitpacking those fields. One uninterrupted sequence needs only one segment; irregular row order can eliminate the benefit. |
| [`StorageDecimal`](storage/storage-decimal.go) | Numbers represented by a shared power-of-ten scale, such as prices in cents or integers sharing trailing decimal zeroes. Stores scaled integers through `StorageInt` plus one exponent. | **64/b×** for the scaled integer payload. Values from 0.00 to 99.99 in cent steps need 14 bits, giving about **4.6×** versus float64. Wide scaled ranges reduce the gain. This is a storage encoding, not an arbitrary-precision SQL decimal implementation. |
| [`StorageFloat`](storage/storage-float.go) | Floating-point data without a suitable scaled-integer representation. Stores float64 values, using NaN for NULL. | **8 bytes/row: 1×** versus a float64 array, or **2×** versus generic 16-byte value slots. No additional floating-point payload compression. |
| [`StorageSparse`](storage/storage-sparse.go) | Mostly NULL columns. Stores only non-NULL values as `Scmer` slots, with bit-packed row IDs. | For non-NULL fraction `p` and `b` bits per row ID, RAM payload is approximately `p × (16 + b/8)` bytes per logical row, plus referenced objects. At 1% non-NULL and 20-bit row IDs this gives about **86×** versus dense 16-byte slots. The disk format uses JSON row-ID/value pairs, so this RAM ratio does not describe the serialized size. |
| [`StorageSCMER`](storage/storage-scmer.go) | General or mixed Scheme values for which a specialized representation is unsuitable. Keeps typed values directly. | **1×** versus generic value slots: 16 bytes per slot plus referenced payloads. This is the fallback, not a compressing codec; JSON serialization has a separate, data-dependent size. |

### Low-cardinality and string columns

| Type | Suitable data and representation | Attainable compression / size model |
| --- | --- | --- |
| [`StorageEnum`](storage/storage-enum.go) | Up to eight distinct values, including NULL, with skewed frequencies: for example, a status flag that is almost always false. Uses rANS entropy coding and a two-level jump index. | Can use **less than one bit/row**. A 99:1 binary distribution has an entropy floor of about **0.081 bits/row**, but actual storage also includes quantized probabilities, chunk boundaries, indexes and the symbol table. For `C` encoded chunks and `G` jump-index groups, the payload plus jump index is `10C + 4G` bytes; the ratio against 8-byte values is **8n/(10C + 4G)×**, before the symbol table and fixed metadata. The entropy floor alone is not an achieved compression ratio. |
| [`StorageString`](storage/storage-string.go) | Repeated text uses a dictionary and bit-packed row IDs. High-cardinality text can use a direct byte buffer with packed starts and lengths. Compatible alphabets receive additional format encoding (below); dictionary bytes can also be LZ4-compressed. | With `n` rows, `k` distinct strings averaging `L` bytes and `b` bits per dictionary ID, the basic payload ratio is **nL/(kL + nb/8)×**, before starts/lengths. For 100,000 rows repeating 100 strings of 20 bytes and 7-bit IDs, this is about **22.3×**. Unique raw text has approximately **1×** text payload plus offset/length overhead; LZ4 gains depend on repeated byte patterns. |
| [`StoragePrefix`](storage/storage-prefix.go) — experimental, automatic selection disabled | Splits common prefixes from suffixes and stores prefix IDs separately. Intended for paths, URLs and similarly structured text. `StorageString.proposeCompression` disables selection pending proper prefix serialization/deserialization. | **No production compression rate claimed.** The model is original text bytes divided by prefix-dictionary bytes, packed prefix IDs and encoded suffix storage. Long shared prefixes offer savings; short or unrelated prefixes do not. This registered implementation is not currently an automatically selected persistent format. |

`StorageString` can preserve the original spelling while reducing the text
payload. Format selection requires compatibility across the column's non-NULL
values; these rates exclude row IDs, offsets and lengths:

| String data | Encoded payload | Compression versus original text |
| --- | --- | --- |
| Lowercase or uppercase hexadecimal; supported numeric, phone/DTMF and date/time alphabets | Four bits per character, densely packed | Approximately **2×** (50% saved), with final-byte rounding. |
| Canonical 36-character UUIDs with consistent lowercase or uppercase spelling | 16 bytes per UUID | **2.25×** (55.6% saved). |
| Supported standard or URL-safe Base64, padded or unpadded | Decoded bytes; the format records how to reconstruct the text | Approximately **1.33×** (25% saved); exact short-string ratios depend on padding and length. |
| Other text, including incompatible mixtures of otherwise compressible alphabets | Raw bytes, potentially shared through the dictionary and compressed with LZ4 | **1×** from alphabet encoding alone; repetition determines any additional savings. |

These are representation changes, not date parsing or case normalization.
See [exact string formats](storage/base64-string-formats.md) and
[compressed-string comparisons](storage/cstring-comparisons.md).

### Blob and computed-column storage

| Type | Suitable data and representation | Attainable compression / size model |
| --- | --- | --- |
| [`OverlayBlob`](storage/overlay-blob.go) | Large strings/binary payloads. Replaces inline contents with content-addressed references, deduplicates identical blobs and gzip-compresses blob contents. The reference column has its own underlying encoding. | For `n` equal blobs of `L` bytes, stored blob size `Z` and encoded reference-column size `R`, the payload ratio is **nL/(Z + R)×**. Repeated, compressible documents can save substantially; unique already-compressed or encrypted bytes are near **1×** and can grow with gzip/reference overhead. Externalizing a blob alone is not compression. |
| [`StorageComputeProxy`](storage/compute_proxy.go) | Derived columns with lazy computation, cached values and dependency invalidation; ordered computations may need to recompute a dependency range. | **No independent codec ratio.** Savings come from avoiding unnecessary materialization and from the representation of cached results. Cache and invalidation metadata cost additional memory; fully materializing a proxy need not be smaller than a plain value column. |

The [format registry](storage/storage.go) lists these implementations;
[format tests](tests/storage/formats/) exercise their behavior. Encodings can
compose, but their quoted ratios cannot simply be multiplied: dictionary and
blob compression act on shared payloads, while row references remain separate.

## Internal storage operators

These are physical Scheme/runtime primitives used by generated plans, not SQL
keywords or a promise of a stable external ABI. The planner selects their
composition. Consult `(help)` and the linked declarations for exact arguments.
A **RecSet** is a query-local set of physical record identities, not an ordered
result. A **RecMap** maps source records to optional target records. Neither may
be persisted or retained across queries and shard-generation changes.

### Scans and ordered execution

Declarations: [storage.go](storage/storage.go), except the separately linked join operator.

| Operator | What it does / typical use |
|---|---|
| `table` | Resolves schema/table names to a storage handle for subsequent operators. |
| `scan_boundary` | Constructs immutable physical access metadata describing a scan's filter/projection boundary. |
| `scan` | Unordered parallel filtered reduction over a table; combines row contributions with the supplied reducer. |
| `scan_batch` | Scan/reduction with batch-backed `#N` pseudo-columns, allowing invocation data to participate without per-row external lookups. |
| `scan_lookup` | Executes an exact-prefix access path from a compiled access schema and flat runtime values; useful for repeated selective probes. |
| `scan_exists` | Tests whether any visible matching row exists without setting up general map/reduce output. |
| `scan_count` | Fused COUNT(*) accumulator helper used by the planner; it increments the count and is not itself a table-scanning entry point. |
| `scan_order` | Produces ordered rows with parallel filtering and serial reduction, supporting ordered limits and order-sensitive consumers. |
| `scan_order_multi` | Merges ordered streams from multiple tables into one sorted reduction stream. |
| `scan_join_order` | Executes a left-deep equi-join in final order, applying global or partition-local OFFSET/LIMIT to joined rows. [Declaration](storage/scan_join_order.go). |
| `scan_order_recset` | Selects the exact ordered OFFSET/LIMIT window but returns its membership as an unordered RecSet; projection/order restoration can happen later. |
| `scan_order_batch_accept` | Examines ordered candidate batches, applies an exact-subset RecSet filter before OFFSET/LIMIT, and grows batches when too few rows survive. |

### Record sets and mapped records

Declarations: [storage.go](storage/storage.go). Implementations:
[RecSet](storage/recset.go), [RecMap](storage/recmap.go).

| Operator | What it does / typical use |
|---|---|
| `scan_recset` | Builds matching record membership, or narrows an existing RecSet without revisiting rows outside it. |
| `recset_count` | Counts the record identities stored in a RecSet. |
| `recset_union` | Unions same-table membership and removes duplicate record identities. |
| `recset_intersect` | Keeps records present in all supplied same-table sets. |
| `recset_difference` | Removes membership of subsequent same-table sets from the first. |
| `recset_not` | Complements membership against the visible rows of the base table. |
| `recset_project_join` | Projects source membership through join keys into a target-table RecSet. |
| `recset_key_index` | Builds immutable composite-key membership lookup over a RecSet. |
| `scan_recmap` | Builds source-to-target record mapping from a filtered table/RecSet using a batch mapper; targets can be absent. |
| `recmap_equi_first_mapper` | Batches equality-correlated zero-or-one target lookups, sharing target work across identical source keys. |
| `recmap_equi_first_of_mapper` | Performs zero-or-one lookups across alternative target-key layouts. |
| `recmap_hash_first_of_mapper` | Evaluates computed target-key alternatives once per target row and uses hash lookup for source batches. |
| `recmap_range_first_mapper` | Finds an ordered range target within equality-key partitions. |
| `recmap_value_mapper` | Materializes reached target values in shard batches and returns a source-record lookup. |
| `recmap_extend` | Creates an extended mapping using original-source and already reached target columns. |
| `recmap_image` | Extracts distinct non-NULL target membership as a RecSet. |
| `recmap_compose` | Chains compatible mappings while retaining the first mapping's source domain. |
| `recmap_order_recset` | Selects an exact source-record window ordered by source or mapped-target columns. |

These mappings do not replace a general many-to-many join. In particular,
first-target lookup is only valid where the planner establishes the appropriate
scalar/ordered-selection semantics. See [operator tests](tests/execution/operators/)
and [RecMap tests](storage/recmap_test.go).

### Reuse, statistics and maintenance

Declarations: [storage.go](storage/storage.go).

| Operator | What it does / typical use |
|---|---|
| `newcachemap` | Creates an evictable, thread-safe key/value cache with shared production of missing values. |
| `initialize_cache_table` | Registers maintenance and initializes a canonical planner cache once against a consistent source snapshot. |
| `cache_table_ready?` | Checks initialization state without building the cache or waiting for it. |
| `touch_keytable` | Extends a cached grouping table's eviction lease. |
| `invalidatecolumn` | Marks computed-column values stale. |
| `invalidateorc` | Invalidates ordered computed values from the affected sort position onward. |
| `scan_estimate` | Estimates scan output cardinality for physical costing. |
| `scan_selectivity_estimate` | Uses learned selectivity or bounded index/sampling estimates, keeping feedback coverage distinct from candidate bounds. |
| `table_planner_statistics` | Reads an immutable planner-statistics snapshot. |
| `table_planner_statistics_token` | Identifies a statistics dependency for cached planning work. |
| `table_planner_statistics_fingerprint` | Summarizes coarse cost classes used for plan reuse. |
| `table_planner_statistics_compatible?` | Checks whether cached statistics remain in a compatible cost class. |
| `table_read_version` | Gives a conservative data-version witness, or nil while mutation is active; enables validated reuse of ordered-cut observations. |
| `table_cache_generation` | Identifies the data generation of an evictable Cache-engine table. |
| `table_shard_count` | Reports active shard count for physical costing. |
| `table_order_partitioned?` | Checks whether partition topology matches the leading order dimension. |
| `table_empty?` | Checks whether a table has rows. |
| `compile_scan_computed_index` | Binds runtime constants into a planner-validated computed-access expression. |
| `shardcolumn` | Suggests partition pivots from column values. |
| `partitiontable` | Applies an initial partition scheme or scores a proposal for later repartitioning. |
| `rebuild` | Rebuilds column storage, merging main/delta state; can target one table. |
| `insert` | Inserts datasets through the storage mutation interface. Generated update/delete paths also use transaction-aware scan/mutation machinery. |
| `blob_inventory` | Audits referenced blobs against backend objects without deleting data. |
| `blob_cleanup` | Deletes objects proven unowned by a complete ownership check; an incomplete check does not authorize deletion. |

Catalog/DDL primitives such as `createdatabase`, `createtable`, `createcolumn`,
`createkey`, `createforeignkey`, `altertable`, `altercolumn`, `renametable`,
`droptable` and their corresponding removal operations underpin SQL schema
changes. Transaction entry points (`tx_begin`, `tx_begin_acid`, `tx_commit`,
`tx_rollback`, `with_autocommit`) are declared in
[transaction.go](storage/transaction.go). These are control operations, rather
than additional scan strategies.

## Scheme runtime and JIT

### Scheme runtime

The embedded runtime provides lexical procedures/closures, pattern matching,
lists and associative values, higher-order map/reduce operations, parser
combinators, strings/regular expressions, JSON, date/time functions, module
loading and native Go builtins. Sessions, promises and synchronization primitives
support shared application state. Local `set` bindings do not mutate an outer
lexical scope.

Sources: [runtime](scm/scm.go), [special forms](scm/special_forms.go),
[parser runtime](scm/parser.go), [Scheme regression suite](lib/test.scm).

### Experimental native compilation

The JIT requires the patched Go compiler's `runtime/jit` integration and
`GOEXPERIMENT=jit`. Ordinary Go builds retain the interpreter. Unsupported
compilation shapes can use interpreter execution; successful SQL parsing does
not mean every expression becomes native code.

| JIT feature | Scope / purpose | Evidence |
|---|---|---|
| Procedure compilation | `jit`, compilation-state inspection and fallback diagnostics; native entry points retain their owning code and constants. | [Compiler](scm/jit.go), [expression tests](scm/jit_expression_test.go) |
| Expression and control-flow lowering | Literals, variables, arithmetic/comparisons, branches, special forms and supported pattern/list operations. | [Expression lowering](scm/jit_compile.go), [special forms](scm/jit_special_forms.go) |
| Closures and recursion | Captured environments, bound closure contexts, direct procedure calls and tested recursive/nested-call forms. | [Compiler](scm/jit.go), [expression tests](scm/jit_expression_test.go) |
| Native/Go call boundaries | Calling conventions for Scheme values, scalar and variadic helpers, argument spills and return values. | [Emitter](scm/jit_x86_emitter.go), [storage ABI tests](scm/jit_storage_abi_test.go) |
| Register and stack allocation | Register ownership, spilling, parallel moves, stack-root tracking and architecture-specific instruction selection. | [Common emitter](scm/jit_emitter_common.go), [selection tests](scm/jit_instruction_selection_amd64_test.go) |
| Parser specialization | Compiled parser combinators, actions, repeated accumulators and literal-leaf paths. | [Parser compiler](scm/jit_parser.go), [parser tests](scm/jit_parser_test.go) |
| Regular expressions | Native lowering for supported matching/replacement patterns, captures and backtracking cases. Other patterns retain fallback paths. | [Regex compiler](scm/jit_regex.go), [native tests](scm/jit_regex_native_test.go) |
| Strings and collections | Specialized string/list operations and batch/vector cases. | [String boundary tests](scm/jit_string_boundary_gojit_test.go), [vector tests](scm/jit_vector_gojit_test.go) |
| Storage specialization | Native storage readers and operator callback ABIs, including compressed-column decoding paths. | [Storage emitter](scm/jit_storage_amd64.go), [storage JIT tests](storage/storage_jit_test.go) |
| GC and stack integration | Precise safepoint maps, stack growth/relocation, retained roots, native callee dependencies and runtime registration. These are implemented mechanisms, not a claim that all GC races have been eliminated. | [Runtime bridge](scm/jit_unwind_gojit.go), [stack tests](scm/jit_stack_gojit_test.go) |
| Publication and code lifetime | Arena allocation, deferred compilation and stackmap/dependency readiness before shared publication; reference retention and cleanup manage native code lifetime. | [Arena/compiler code](scm/jit.go), [pool tests](scm/jit_pool_test.go) |
| Panic and cancellation paths | Tested panic propagation across native frames and cancellation of compilation through its execution scope. | [Expression tests](scm/jit_expression_test.go), [stack/cancellation tests](scm/jit_stack_gojit_test.go) |
| Diagnostics | Code dumps through `MEMCP_JIT_DUMP_DIR`, optional Linux perf maps and compilation/fallback logging. | [Compiler diagnostics](scm/jit.go) |

**Architecture scope:** amd64 has the broad expression/parser/storage lowering
path. ARM64 and RISC-V have architecture emitters and native portable-leaf tests;
this is not feature parity with amd64. Other architectures disable JIT support.
See [build gating](scm/jit_feature_enabled.go),
[portable-leaf tests](scm/jit_portable_leaf_gojit_test.go) and the
[JIT build instructions](README.md#build-with-the-experimental-jit).

## Operations, security, and data movement

| Area | Capabilities / entry points |
|---|---|
| Authentication and permissions | Users, passwords, grants/revokes and database access policies; HTTP and MySQL authentication. [Tests](tests/sql/security/). |
| Dashboard | Query activity, process inspection, storage/compression, users, settings, logs and metrics. [Dashboard](lib/dashboard.scm). |
| Runtime settings | Memory budgets, diagnostics and other server controls, with persisted settings. [Implementation](storage/settings.go). |
| Query control | Process lists, session identities, `KILL QUERY` / `KILL CONNECTION` and transaction cancellation checks. [Tests](tests/execution/concurrency/processlist.yaml). |
| Memory and maintenance | Eviction budgets, maintenance capabilities, rebuilds and repartitioning, blob ownership audit/cleanup. [Storage API](storage/storage.go). |
| Failure notifications | Named asynchronous persistence-failure callbacks, cooldown/coalescing and dashboard-managed persistent definitions. [Hooks](lib/storage-failure-hooks.scm). |
| Live imports | MySQL and PostgreSQL schema/data import; selected target tables are replaced, not continuously replicated. [MySQL](storage/mysql_import.go), [PostgreSQL](storage/psql_import.go). |
| Backend migration | Database storage can be moved through `alterdatabase_storage`, publishing the new configuration after transfer. [Migration tests](storage/persistence_move_test.go). |
| File imports | SQL dumps, supported PostgreSQL dump/archive inputs, CSV, JSONL and RDF/Turtle. [Import tests](tests/integration/import/), [storage loaders](storage/storage.go). |
| Backup and recovery validation | Dump/restore checks, crash/restart tests, fault injection and multi-shard transaction recovery drills. A tested backup/restore procedure is still required for an actual deployment. [Persistence suites](tests/storage/persistence/), [drill guide](tools/reliability-drill.md). |
| Distribution | Source builds, PHP/no-PHP and JIT variants, DEB/RPM packaging, containers and release artifact checks. [Build targets](Makefile), [deployment guide](README.md#installation-packages). |

## Packaging and distribution

The release workflow builds native Linux packages, a standalone binary and
container images. Package and container builds have different runtime contents:

| Distribution | Capabilities and contents | Build / source |
| --- | --- | --- |
| **DEB** — Debian/Ubuntu | Release artifact for **amd64**. Includes embedded PHP with its matching ZTS runtime, PHP configuration and Imagick module, plus a dedicated service account and systemd service. | `make memcp.deb`; [Debian packaging](debian/). |
| **RPM and source RPM** | Release artifacts for **x86_64**, built and tested on Fedora. Includes the same embedded-PHP runtime and service integration as the DEB. The source RPM is independently rebuilt in CI using the declared external PHP SDK. | `make memcp.rpm`; [RPM spec](memcp.spec), [native RPM workflow](.github/workflows/rpm.yml). |
| **Docker / OCI image** | `carli2/memcp` with version and `latest` tags, supporting **linux/amd64 and linux/arm64**. Uses a minimal Alpine runtime and a static Go build without embedded PHP. Includes Scheme modules, assets, an HTTP health check and a persistent `/data` volume. Runs as unprivileged UID/GID **10001:10001**. | [Dockerfile](Dockerfile), [entrypoint](packaging/docker-entrypoint.sh); release builds use Docker Buildx. |
| **Standalone Linux binary** | Static **amd64** release artifact, built explicitly with `nophp`; no embedded PHP runtime. | [Release workflow](.github/workflows/release.yml). |

Native packages use `/etc/memcp/memcp.conf` for configuration,
`/var/lib/memcp` for data and `/run/memcp/memcp.sock` for the local MySQL socket.
First installation creates a root-only initial-password file at
`/etc/memcp/initial-root-password`. Upgrades stop the service gracefully and
restart it after replacement; removal preserves database data. The systemd
service restricts writable paths to the data and runtime directories.

Containers accept the initial password through
`MEMCP_ROOT_PASSWORD_FILE` (default `/run/secrets/memcp_root_password`) or
`ROOT_PASSWORD`. Initialization applies only to a fresh data directory;
subsequent starts retain stored credentials. Docker Compose deployments can
combine the data volume with a mounted secret. See the
[installation and deployment examples](README.md#installation-packages).

Local DEB/RPM builds write artifacts to `dist/`; `make package-check` inspects
package contents. Release validation also runs Lintian and RPM integrity
checks, and the native RPM workflow exercises installation, upgrade, removal
and the packaged PHP runtime. Tagged releases publish **SHA-256 checksums and
build-provenance attestations** for release artifacts, and **provenance and an
SBOM** for container images. The workflow verifies that the `vMAJOR.MINOR` tag
matches `CHANGELOG.md` and belongs to `master` before publishing.

Sources: [build and packaging targets](Makefile),
[package checks](tools/test_packaging.py),
[release workflow](.github/workflows/release.yml).

## Validation and development tools

- Scheme startup tests, Go unit tests and categorized YAML integration tests.
- Must-succeed and must-fail SQL cases, concurrency tests and restart/recovery
  suites, including compatibility checks against historical release writers.
- Performance A/B comparisons against a PR's base, with common fixtures and
  sampling, plus separate JIT correctness/performance workflows.
- Query EXPLAIN stages, operator-level measurements and generated cost models.
- Scheme formatting checks, SQL regression-policy checks and test-runner
  contract tests.
- Generated runtime/API documentation through `make docs`; generated output is
  not versioned. This catalog supplies the overview, not a duplicate signature
  reference for every builtin.

See the [test taxonomy and measurement rules](tests/README.md),
[CI workflows](.github/workflows/), [developer guidance](AGENTS.md) and
[planner invariants](INVARIANTS.md). Claims about an entire workload should be
backed by its schema, query/result checks, engine mode and measured execution;
a feature name alone does not establish compatibility or performance.
