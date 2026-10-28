<!-- Copyright (C) 2026 Carl-Philip Hänsch -->

# UUID support and cluster INSERTs

UUID columns validate assignments and use canonical lower-case UUID text at
frontend boundaries. In execution and rebuilt storage they use the existing
immutable 16-byte CString carrier. All UUID versions, including the nil UUID,
can be stored. Text columns preserve their existing spelling and historical
storage formats; no disk-format identifiers change.

Matching packed UUIDs compare directly as 16 bytes. Hashing formats into local
stack scratch in the same hash domain as plain text, without a string allocation.
Scalar and batch UUID reads retain packed values. The outer LZ4 container still
needs decoding during reader preparation. JSON, WAL and network serialization
can use text; this is not a separate opaque Scheme type or a binary-only wire
protocol.

## Functions

The existing `(uuid)` remains a v4 string generator. New Scheme functions:

| Function | Result |
| --- | --- |
| `(uuid_v1)`, `(uuid_v4)`, `(uuid_v7)` | Packed UUID |
| `(uuid_parse value)`, `(uuid_format value)` | Canonical packed UUID / text |
| `(uuid_valid value)` | Parse validity; NULL propagates |
| `(uuid_to_bin value [swap])`, `(uuid_from_bin bytes [swap])` | 16 binary bytes / packed UUID |
| `(uuid_version value)`, `(uuid_variant value)` | Version bits / variant name |
| `(uuid_timestamp_ms value)` | Unix milliseconds embedded in v1/v6/v7, otherwise NULL |
| `(uuid_timestamp value)` | UTC timestamp text with millisecond precision |

Conversions propagate NULL and reject malformed non-NULL input. Binary input
must contain exactly 16 bytes. Swap flag 1 applies the MySQL v1 permutation;
conversion back must use the same flag. Flag 0 preserves RFC byte order.

| Frontend | Supported surface |
| --- | --- |
| MySQL | `UUID()` (v1), `IS_UUID`, `UUID_TO_BIN`, `BIN_TO_UUID`; `UUID_V4()` and `UUID_V7()` extensions |
| PostgreSQL | `uuid`, `::uuid`, `CAST(... AS uuid)`, `gen_random_uuid()`, `uuid_generate_v4()`, `uuidv4()`, `uuidv7()` |
| T-SQL | `uniqueidentifier`, `CAST`/`CONVERT`, `NEWID()` (v4), `UUIDV7()` extension |

MySQL/PostgreSQL also expose `UUID_EXTRACT_VERSION`, `UUID_EXTRACT_TIMESTAMP`
and `UUID_TIMESTAMP_MS`. The version extractor returns NULL for non-RFC variants.
Timestamp text preserves milliseconds; the ordinary date carrier has only second
precision. PostgreSQL's optional `uuidv7(interval)` shift is not implemented.
SQL Server's Windows-specific `NEWSEQUENTIALID()` is not aliased to a different
algorithm. T-SQL UUID ordering currently follows canonical byte order rather than
SQL Server GUID field ordering. INSERT `RETURNING` / `OUTPUT` is not added.

## Generated keys

```sql
CREATE TABLE events (
    id UUID NOT NULL DEFAULT UUIDV7() PRIMARY KEY,
    payload TEXT
);
INSERT INTO events(payload) VALUES ('first'), ('second');
```

T-SQL uses `UNIQUEIDENTIFIER` and a suitable payload declaration, for example
`NVARCHAR(100)`. MySQL also accepts `DEFAULT (UUID_V7())`.

The SQL regression suites also execute the conventional server syntax directly:

```sql
-- MySQL: 16 binary bytes, version-one UUID generator
CREATE TABLE uuid_mysql_official (
    id BINARY(16) DEFAULT (UUID_TO_BIN(UUID())) PRIMARY KEY,
    payload TEXT
);
-- PostgreSQL: native UUID type, version-four generator
CREATE TABLE uuid_postgresql_official (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    payload text
);
```

These cases check single-row INSERT, multi-row INSERT and INSERT SELECT,
including exact distinct-key counts and binary length/UUID version. The MySQL
expression follows the [documented expression-default example](https://dev.mysql.com/doc/refman/8.4/en/data-type-defaults.html);
PostgreSQL's generator is [documented here](https://www.postgresql.org/docs/18/functions-uuid.html).
Binary UUID defaults retain a closed generator/conversion recipe with swap flag
in native declaration metadata, evaluated per row. WAL/rebuild reload tests
verify the binary key bytes, duplicate detection and further generated inserts.

Omitted UUID defaults are evaluated once per input row before routing, UNIQUE
checks and WAL construction, including `INSERT ... SELECT`. Explicit IDs remain;
explicit NULL follows nullability. Integer AUTO_INCREMENT/IDENTITY stays numeric;
UUID declarations with integer AUTO_INCREMENT are rejected. PostgreSQL/MySQL
ALTER SET DEFAULT preserves generator names instead of sampling at DDL time.

The existing UUID library supplies random bits and local v7 monotonicity. Separate
nodes and restarts do not establish global monotonicity or commit order. Embedded
generation time does not replace an application/commit timestamp column.

## Adaptive partitioning

Rebuild chooses a method separately for each weighted candidate column. UUID
and integer equality/UNIQUE demand can select hash partitioning; range and
ordered demand favor range partitioning. The cost estimate charges hash range
access for every bucket and estimates point-writer contention for range access.
This is advisory feedback rather than a measured throughput prediction. Without
fresh observations, an existing hash method survives restart.

Hash identifiers `hash-uuid-v1` and `hash-int-v1` persist their byte-level mapping
with the topology. UUID hashing uses RFC bytes, integers fixed-width signed bytes.
Equality selects one bucket; ranges and coercions without a compatible domain
scan all buckets. Hash partitions never supply ordered-range proofs. Declarative
`shardcolumn`/`partitiontable` hints retain their range meaning.

## Cluster allocation — draft limitations

**The cluster INSERT implementation is unfinished.** The schema-write allocation
and global UNIQUE gates below violate the required INSERT invariant and must be
replaced before this PR is ready. Passing correctness tests does not make those
gates an accepted concurrency contract.

Free-shard INSERT routing prefers locally owned M/O writer lanes. Newer foreign
shards do not displace older usable local lanes. Missing local capacity creates
a new lane under schema MOSI ownership and durable topology publication. Busy
maintenance cannot make the fallback claim a foreign writer. Range partitions
retain key-based routing.

Integer batches and `reserve_ids` contact the current schema owner before data
write admission and reserve blocks from the existing table counter. Explicit
positive IDs advance the same counter. The owner checkpoints before acknowledging
a remote reservation, so another node's committed IDs cannot be reused after an
allocator restart. There is no permanent seed/master allocator or second sequence
catalog. Interrupted reservations may leave gaps and are not automatically retried.
The existing non-cluster allocation and durability contract are unchanged.

Global INSERT UNIQUE checks currently serialize through existing schema write
admission, after obtaining their read generations and before data write admission.
This still causes schema coordination and remote generation preparation; it is
not a distributed uniqueness index. General cross-node transactions, oversized
remote constraint inputs, collision-callback recursion and broader DML
coordination are outside this implementation's verified contract.

Cluster requests include their originating process incarnation. Request numbers
are ordered within that incarnation; a restarted process's new sequence is not
suppressed by its predecessor. Directory marks and permissions remain RAM-only.
Cluster nodes must run compatible protocol implementations.

Run the three-peer INSERT/restart oracle with:

```sh
python3 tools/replication_cluster_stress.py --cluster-inserts-only --seed 20261027
```

It checks parallel UUID/integer batches, explicit integer high-water marks,
the complete payload multiset on each peer, concurrent duplicate keys, rollback
and restart. It never retries failed INSERTs or infers rights from backend files.
CI runs normal and race builds.

## Integration with current master

This branch incorporates master #1193, #1194 and #1197. Native UUID successor
routing replaces the old lineage/alias assumptions. #1194 supplies the owner
handler/serialization/lifetime foundation; #1197 closes computed preparation's
native read-admission gap, passes the query transaction through acquisition and
drains all started materializations before propagating failure. Those mechanisms
are reused rather than replaced by additional reader locks or copied sessions.

The three-peer SQL INSERT oracle exposed complementary correctness gaps: UNIQUE
probes must use the generation returned by acquisition; invalidation must reach
readers which still know the predecessor UUID; and an installed reader's delayed
grant must not resurrect it after revocation. A separate RAM-only revoked-grant
floor preserves the actual observed owner epoch and unrelated pending flights.
Restart invalidation acknowledges missing permissions without recovering rights
from shared files. These are protocol corrections, not a completed replacement
for the remaining schema-M allocator and UNIQUE gate.

The integrated normal and race three-peer INSERT/restart runs pass with unchanged
oracles and deadlines. Six focused YAML suites cover 229 cases, including 18
crash/WAL/rebuild/restart checks in `tests/storage/persistence/uuid-defaults.yaml`.
The process oracle still exercises free lanes; hash-partitioned distributed
INSERTs and general remote sync effects remain separate required coverage.

## SCM execution boundaries still to complete

SQL `EXPLAIN` on the UUID fixture exposes these actual physical operations:

| Query shape | SCM boundary | Ownership requirement |
| --- | --- | --- |
| COUNT / COUNT DISTINCT | `scan`, mapper/reducer lambdas, `set_assoc_mut`, `merge_assoc_mut` | Each worker owns its reduction state; coordinator combines completed values |
| Maintained group cache | `initialize_cache_table`, `createcolumn`, `group_insert_batches`, `$set:*` callbacks | Cache preparation owns its lifecycle; callbacks cannot become remote storage pointers |
| Ordered row projection | `scan_order`, callback calling `resultrow` | Workers produce ordered batches; result delivery belongs to the requesting executor |
| Generated keys | Native `insert`, `reserve_ids`, first-ID callback | Durable integer allocation and UNIQUE admission must use data ownership without schema M |

`EXPLAIN INSERT` is not supported by the current parser; its native primitives
above are verified in `storage/storage.go`, rather than inferred from SELECT.
Plain ordered remote projection remains unsupported by the current delegation
boundary. The INSERT oracle verifies the full payload multiset through an
unordered string reduction; it does not claim ordered projection coverage.

Master #1194 supplies `ClusterObject` owner handlers, serialization contexts,
peer-scoped lifetimes and session/cachemap views. This PR inherits those mechanisms;
constant captured reads use their invocation view, and cache misses retain the
owner boundary. That does not establish remote mutation contracts for every sync
callable (`once`, promises, mutexes). The remote validator must continue rejecting
unimplemented effects. Shared effects still need batch-level execution,
cancellation and ambiguous completion handling; a network call for every row is
not an acceptable substitute.
