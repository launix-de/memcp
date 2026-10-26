<!-- Copyright (C) 2023 - 2026 Carl-Philip Haensch; GPL-3.0-or-later -->

# MemCP — transactions, search, and analytics in one open-source database

MemCP is a persistent, column-oriented SQL database built for applications that
must update data continuously while also searching, filtering, and analyzing
large datasets. It brings an architecture best known from proprietary in-memory
ERP systems to a self-hosted GPL-licensed database.

MemCP keeps stable data in compact, read-optimized columns and accepts recent
changes in a write-friendly delta. Background rebuilds fold those changes back
into the compressed representation. Persistent data may be evicted from RAM and
loaded again on demand, so the complete database does not have to remain memory
resident.

Applications can connect through the MySQL wire protocol or submit MySQL- and
PostgreSQL-style SQL through separate HTTP endpoints. An initial T-SQL mode adds
`/tsql/<database>` and an optional TDS listener (`--tds-port=1433`). An RDF/SPARQL engine is
included as well.

> **Status: Beta.** MemCP is suitable for evaluation and controlled beta
> deployments. Common application SQL is covered by a large regression suite,
> while advanced dialect edge cases are still being completed. Validate the
> queries, durability configuration, backup procedure, and result parity of
> your application before a production cutover.

MemCP is licensed under GPL-3.0-or-later.

See the [feature catalog](FEATURES.md) for SQL and frontend support, storage
backends and operators, planner capabilities, and the experimental JIT.

Read the [MemCP 0.9 beta release overview](https://launix.de/launix/memcp-beta-release-0-9-one-database-for-transactions-search-and-analytics/)
for the application story behind the current release.

## Why this project exists

During doctoral work at a university database chair, MemCP's initiator kept
encountering the same gap: open-source databases offered either conventional
transaction processing or columnar analytics, but no practical columnar
main/delta in-memory database for an ERP application that needed both. SAP HANA
demonstrated that the model could work at ERP scale, but the architecture
remained tied to a proprietary platform.

MemCP was started to make that database model available as ordinary free
software: run it on your own Linux server, connect an existing SQL application,
and use the same engine for transactions, full-text search, paged lists, and
analytics.

## Why MemCP?

- **Columnar execution:** scans and aggregates read only the columns they need.
- **Automatic indexing:** MemCP derives useful compound indexes from real
  access, filter, and ordering patterns.
- **Cost-based query planning:** the compiler rewrites correlated subqueries,
  reorders joins, and chooses a physical strategy for each part of a query.
- **Adaptive plans:** cached plans are retained while cardinalities remain in
  the same planning range and are reconsidered when relevant statistics cross
  a cost boundary.
- **Mixed workloads:** point access, writes, filtered lists, exact counts,
  full-text-like search, and complex analytics can use the same database.
- **Operational tooling:** packages, containers, imports, persistent storage,
  memory budgets, and an administration dashboard are included.

### MySQL client access

Existing MySQL clients and connectors can use MemCP's wire-protocol server.
SQL compatibility is broad but not yet a claim of complete MySQL equivalence;
test the exact query shapes used by your application.

```sql
CREATE TABLE users (id INT, name VARCHAR(100), email VARCHAR(255));
INSERT INTO users VALUES (1, 'Alice', 'alice@example.com');
SELECT * FROM users WHERE id = 1;
```

### SQL over HTTP

```bash
# Start MemCP with HTTP and MySQL interfaces
./memcp --api-port=4321 lib/main.scm

curl -u root:admin http://localhost:4321/sql/mydb \
  -d 'SELECT * FROM users'
```

Important HTTP endpoints:

- `/sql/<database>` — MySQL-dialect SQL
- `/psql/<database>` — PostgreSQL-dialect SQL
- `/tsql/<database>` — initial T-SQL grammar using the same planner and storage
- `/rdf/<database>` — SPARQL queries
- `/rdf/<database>/load_ttl` — load RDF/Turtle data
- `/dashboard` — administration, system monitoring, query activity, storage,
  compression, and users

`make` and `make jit` build `memcp` with PHP support. Use `make nophp` for a
size-focused build without PHP, or `make jit-nophp` for a JIT build without PHP.
PHP remains an explicit feature: `--serve`
mounts the app into the existing HTTP handler chain and does not create a
second listener.

These are SQL-over-HTTP APIs rather than a resource-oriented REST data model.

T-SQL mode currently covers bracket/double-quoted identifiers, Unicode literals,
TOP, OFFSET/FETCH and basic DDL/DML through native defaults, constraints and
auto-increment. Enable TDS with `--tds-port=1433 --tds-database=<existing database>`.
The transport is the separate [go-tdsstack](https://github.com/launix-de/go-tdsstack)
module. RPC parameters bind through frontend declarations, including binary
bytes and numeric coefficients. Pure SELECT descriptions expose column and
parameter contracts without executing the query. National character declarations
retain their original schema types; assignment and cast width uses UTF-16 units.
Integer division uses the declared operand types and rejects zero divisors.

Schema DDL retains declared identifier spelling while resolving table, view and
column references without case sensitivity. Named defaults use the existing
native column defaults, including ALTER ADD/DROP. CREATE TYPE ... FROM stores
scalar alias definitions in `system.tsql_types`; columns retain their qualified
alias declaration while the frontend resolves expression types, dimensions,
nullability and casts. Alias contracts for RPC/procedure parameters and CAST,
historical column conversions, and complete system-object catalogs remain open.

TDS binary results preserve arbitrary bytes, empty values and NULL. Direct
DECIMAL/NUMERIC columns, parameters and casts expose their declared precision
and scale. Integer expression results retain the native signed 64-bit range
when sent as zero-scale NUMERIC, including values beyond floating precision.
Scaled casts use the existing numeric precision; exact DECIMAL(38,s), computed
decimal precision propagation, temporal parameters, output parameters, and
text/numeric-to-binary encoding remain open. Computed decimal results without
known dimensions currently retain approximate numeric transport. Metadata
discovery remains partial; scripts use the shared request/statement lifecycle.
Non-NULL binary RPC inputs currently require a binary declaration; their native
string carrier is never interpreted as numeric or character text.
T-SQL plans retain the existing plan cache and JIT; the dialect currently uses
exact query text, as its literal syntax differs from the shared lexer.


### Reverse-proxy handlers

`(serveProxy "http://127.0.0.1:9000")` creates a normal `(req res)` handler.
It can be passed to `serve` directly or mounted in the existing handler chain:

```scheme
(define backend (serveProxy "http://127.0.0.1:9000"))
(define http_handler (begin
	(define previous http_handler)
	(lambda (req res)
		(if (regexp_test (req "path") "^/backend(?:/|$)")
			(backend req res)
			(previous req res)))))
```

The upstream URL's base path is joined with the incoming path; the mount prefix
is preserved. Upstream and incoming query parameters are combined. The handler
forwards the method and body, streams the upstream response, and cancels the
upstream request when the client disconnects. Unreachable upstreams return 502.
Hop-by-hop headers are removed and forwarding headers are rebuilt from the
incoming connection. Creating the handler does not contact the upstream.

## Architecture

The storage engine and protocol/runtime integration are implemented in Go. SQL
parsing, logical planning, optimization, and physical-plan generation are
implemented in Scheme.

The planner has an explicit phase boundary:

```text
SQL parser AST
    -> decorrelation and logical normalization
    -> join reorder and logical optimization
    -> cost-based physical lowering
    -> fused storage operators
```

Correlated subqueries are converted into reorderable joins before physical
planning rather than executed once per outer row. The physical planner can then
choose an execution strategy independently for each relevant part of the query.

The engine stores columns in compact representations, including bit-packed
integers, dictionaries, ranges, and sparse forms. Hot indexes and cached query
helpers are memory-managed. Persistent column data remains on disk when its
in-memory representation is evicted.

The optional JIT compiler is experimental. All benchmark ranges below were
measured without enabling the JIT.

## Performance

Performance depends on data distribution, query shape, cache state, durability
mode, and hardware. MemCP is not universally faster than another database.

Current application measurements include:

- **up to 90x faster than PostgreSQL** for selected real application searches
  and paged lists over the same dataset of roughly 800,000 documents. These
  queries combine full-text conditions, sorting, exact counts, pagination, and
  user-specific access checks;
- **from about 3% faster to 15x faster** on selected WordPress-oriented query
  workloads.

These ranges describe the measured workloads, not a blanket guarantee. Publish
or compare results together with the fixture, database configuration, indexes,
hardware, cold/warm state, repetitions, and individual query timings.

## Performance tuning

MemCP's query performance is a property of the cost-based planner, not of
hand-tuned switches. Making a slow query fast means getting the planner to
choose the plan you would write by hand, and — only when the operators
genuinely cannot go faster — improving the operators themselves.

The workflow used in this repository:

1. **Add a perf regression case** for the query under `tests/`, with its setup
   and measurement configuration, then run and measure it.
2. **Inspect the plan.** `EXPLAIN <query>` prints the generated Scheme plan;
   `EXPLAIN IR`, `EXPLAIN REORDER`, and `EXPLAIN PHYSICAL` show the logical IR,
   the chosen join order, and the lowered physical operators with their cost
   estimates.
3. **Measure the parts.** Send the extracted plan fragments to the `/scm`
   endpoint one by one to find where the time goes. `(help)` lists the storage
   operators.
4. **Build the target plan by hand** from different scan variants, RecSets, and
   operator fusions; measure until you have the optimal plan.
5. **Make the planner reach it.** Review the planner, the physical lowerer, and
   `tools/costgen`, then refactor so the hand-built plan is inside the search
   space and is estimated as the cheapest — calibrate its cost against the
   competing operators rather than special-casing it.
6. **Introduce a new storage operator only when the existing ones are
   exhausted.** It must be integrated into the planner and cost-calibrated
   against the alternatives, not bolted on beside them.

`tools/costgen` runs the tagged YAML workloads through every forced physical
alternative, validates the results and operators, solves a non-negative cost
equation system from the measurements, and regenerates the planner's cost
constants.

Every pull request is compared against its exact base revision by a CI A/B
benchmark using the same fixtures, warm-up, and repetitions; no protected case
may regress more than 20%. Run the local perf suite with `PERF_TEST=1 make
test`. Performance PRs must additionally include manual A/B measurements
(development baseline vs. change, same fixture and sampling) in the description.

## Docker

```bash
docker pull carli2/memcp
printf '%s\n' 'replace-with-a-long-random-password' > memcp-root-password.txt
chmod 600 memcp-root-password.txt
docker compose up -d
```

The container intentionally refuses to initialize a fresh data volume without
an explicit password. See [Container deployment](#container-deployment) for the
complete Compose configuration.

## Memory management

MemCP is designed to share a machine with other services while remaining within
an explicit RAM budget.

**Automatic compression** — MemCP selects compact representations per column.
Actual compression depends on the values and their distribution; benchmark it
with representative data rather than assuming a fixed compression ratio.

**Configurable memory budget** — by default MemCP's managed cache budget is 50%
of system RAM. Set an exact limit through the dashboard or Scheme API:

```bash
# Limit to 4 GB total
curl -u root:admin -X POST http://localhost:4321/scm \
  -d '(settings "MaxRamBytes" 4294967296)'

# Or as a percentage of total RAM (default: 50)
curl -u root:admin -X POST http://localhost:4321/scm \
  -d '(settings "MaxRamPercent" 40)'
```

**Automatic eviction** — when MemCP approaches its memory limit, it unloads
eligible, less-recently-used representations from RAM. Persistent data remains
on disk and is transparently loaded when a later query needs it.

`MinRamBytes` sets a lower bound on CacheManager-tracked memory (bytes, default
`0` = disabled). Set it with `(settings "MinRamBytes" 1073741824)` to protect
1 GiB from automatic eviction, including system RAM pressure, persisted/total
budgets, proactive allocation pressure, and idle expiry. Indivisible eviction
offers that would cross the floor are skipped. The minimum takes precedence
over a smaller maximum budget. It does not reserve RAM or measure process RSS;
explicit removal/invalidation and ordinary object lifetimes can reduce usage.

**System-wide pressure awareness** — when the host runs low on available RAM,
MemCP can release cache entries even if its configured budget has not yet been
reached.

**Separate budget for persistent data** — a second managed budget (default: 30%
of system RAM) controls persisted shards and indexes independently of other
evictable query helpers. Both limits are tunable at runtime without restart.

### Storage engines

MemCP supports several storage engines, selectable per table via `CREATE TABLE ... ENGINE=<engine>` or `ALTER TABLE ... ENGINE=<engine>`:

| Engine | Durability contract | Evictable |
|--------|---------------------|-----------|
| `safe` | **Default.** WAL is fsynced at transaction/statement boundaries; committed writes survive process crashes and power loss. | Yes |
| `logged` | WAL is written without fsync. It survives a process crash or clean shutdown, but recent writes may be lost after power loss. | Yes |
| `sloppy` | Compressed column files are persisted, but there is no WAL. Deltas since the last rebuild are lost after an unclean shutdown. | Yes |
| `memory` | Rows exist only in RAM and are lost on restart. Schema and an optional reconstruction callback are persisted. | No |
| `cache` | Reconstructible RAM data; it may be cleared under memory pressure and rebuilt by its initializer. | Yes |

> **Data-safety warning:** `ALTER TABLE ... ENGINE=memory` from a persisted
> engine irreversibly deletes that table's on-disk column files and WAL. Back
> up or otherwise verify the data is disposable before making this transition.

For production data, use `safe` unless you have explicitly accepted another
engine's weaker durability contract.

`ALTER TABLE ... ADD COLUMN ... DEFAULT ...` initializes existing rows with
the value captured by that declaration. Later `SET DEFAULT` or `DROP DEFAULT`
changes future omitted INSERT values; it does not rewrite existing values or
explicit NULLs. A populated table cannot add a `NOT NULL` column without a
non-null default. ADD supports the frontend's literal defaults, constant
arithmetic/casts where supported, and `CURRENT_TIMESTAMP`; other default
expressions fail before the column is published.

### Experimental local replication

Run a replication master with `--replication=listen` and a slave with
`--replication=ws://master-host:4321/cluster`. The master serves the `/cluster` WebSocket
endpoint on its ordinary HTTP API listener; no separate port is needed.
`--replication=listen` requires an enabled HTTP API listener. Explicit
HTTP(S)/WS(S) URLs may include a proxy path; a URL without a path uses `/cluster`.
Secure URLs use the normal TLS certificate validation. The built-in HTTP listener
currently serves plain HTTP; HTTPS/WSS requires a TLS endpoint in front of it.
The optional dedicated listener remains available with `--replication=7331`
and `--replication=master-host:7331`; it uses exactly the same node protocol.

`lib/main.scm` calls `replication_start` and constructs the listeners and routes.
The handler returned by `node_protocol` is mounted explicitly, for example:

```scheme
(serve port
	(http_route "/engine/node" (node_protocol)
		(lambda (req res) (http_handler req res))))
```

The standard startup script fixes the route at `/cluster`, like `/dashboard`.
The example above is for a custom Scheme server; its peers would use
`--replication=ws://master-host:4321/engine/node`. Other HTTP servers do not
inherit the route. The native protocol handler owns its own authentication;
the Scheme fallback retains ordinary HTTP authentication.

Each instance needs its own `-data` directory
and independent backend namespace, including separate buckets/prefixes for
remote storage. The existing persistence backends are used without modification.
`schema.json` remains the schema format. `SHUTDOWN` affects the addressed instance.

Both listeners negotiate `memcp.mosi.v1` and exchange bounded binary WebSocket
messages. Each message contains a versioned envelope, JSON control metadata and
raw compressed transfer bytes, without base64 expansion of shard chunks. Text
messages, unsupported versions and oversized messages are rejected. Older JSON
replication peers must be upgraded together. SQL requests on the same HTTP port
continue to use their ordinary authentication; `/cluster` owns its existing
replication authentication and per-message authorization.
The shared route dispatches the explicit `X-MemCP-Role` to its enabled adapter;
selecting a role grants no rights, and removing one adapter preserves others.

The same endpoint also supports authenticated remote shard reductions (see
below). Cluster membership and automatic placement remain separate from this
operator. Cluster peers are equal participants: the first node and the connect
address confer no permanent coordinator role. A graceful cluster
leave must hand over MOSI rights and directory duties before stopping, allowing
clients to reconnect to a second node and the first node to be removed.

HTTP and MySQL logins on the slave are verified by the master and cached for
offline authentication. A slave claims the opened database, retains durable
ownership through disconnects/restarts, and can use its local `SAFE` tables
while the master is offline. An unavailable remote owner produces
`Cluster network error`; disconnection never transfers ownership. Use `SAFE`
for crash recovery: replication does not strengthen `memory` or `sloppy` durability.

The dashboard Settings page lists slaves and offers an explicit **Retire slave**
action. Retirement makes the last master backup authoritative and may lose
untransferred offline changes. The persisted `ReplicationUUID` is also the
slave's secret; do not share or clone its settings between installations.

Each database schema and each logical data shard has an independent MOSI
slot. The initial connection acquires schema M, discovers its shards, then
requests shard M through the ordinary access protocol. Later reads request S;
M becomes O at its previous owner. Writes acquire M and invalidate other copies.
The MR/OR/SR states coalesce concurrent requests and allow read-to-write upgrades.

Full offline operation, including DDL, requires M for the database schema and
all its data shards. Shared copies still permit reads; writes require M for
the affected resource.
Master analytics can leave those resources shared; they do not automatically
return to M when the query ends. A later write reclaims M through the ordinary
protocol while the other holders are reachable. On the slave, an administrator
can run `(replication_reclaim "database")` through `/scm` to acquire M for its
schema and every active durable shard, including empty and internal shards.
It returns true only if all those permissions are still exclusive at the final
check. Subsequent remote queries can share them again. An unreachable holder
causes `Cluster network error`; reclamation never overrides its ownership.

Before each global rebuild, the existing 15-minute maintenance job also attempts
this reclaim for every fully initialized database on a slave. Failures are logged
per database; other databases and the rebuild still proceed. It does not subscribe
new databases or reclaim on a master. The next run is scheduled 15 minutes after
the previous job finishes, so long rebuilds extend the interval. Later remote reads
can share permissions again; periodic reclamation is not an offline-write guarantee.

IS denotes a current backend-only generation with durable local permission;
IO must acquire from its remote owner even when an older backup exists locally.
Loading IS restores its existing permission and cannot turn a shared copy into M.
Already authorized reads check atomic state without additional replication locks,
reader counters or registration. Existing snapshots retain historical generations.
Only writes participate in the handover gate, including explicit transactions.

Transfers stream independently compressed chunks through bounded queues into
backend writers. Handover rebuilds exactly the requested shard; receivers install
its generation and point topology update before acknowledging or releasing waits.
Slave admin SQL runs against the master's own authentication and access catalog;
SHUTDOWN remains local. Multi-master routing and distributed scans remain future
work. The separate **Replication Cluster Stress test** workflow runs a Python
process/thread runner with alternating process crashes, parallel shared reads,
independent partition ownership, complete database reclamation, offline writes
and durable restart. Two simultaneous slaves claim different databases while a
third, master-only database runs schema changes and verified data queries.
Quiescent checkpoints compare complete current values on both nodes. These tests
retain data directories; retiring a destroyed owner's disk can lose changes newer
than its last transferred backup.

Run it after building MemCP with
`python3 tools/replication_cluster_stress.py --seed 20261023`.
It owns independent master/slave processes, uses HTTP and the `mysql` client,
and retains node logs, a write oracle and the seed in its printed artifact path.
`--rounds`, `--workers` and `--operations` control the threaded workload.
The runner uses the normal HTTP `/cluster` route by default;
`--dedicated-replication-port` tests the optional dedicated listener.
Go tests cover the in-process MOSI and transport units separately.

The coherence state machine lives in `storage/mosi`, independently of storage,
SQL, transport and backend policy. Its public API exposes immutable permission
snapshots and request completion. Engine framing and connections are in
`storage/mosi_wire.go` and `storage/mosi_transport.go`; replication directory
coordination, generation transfer, authentication, configuration, session integration
and startup use the corresponding `storage/replication_*.go` files. This separates
the shared coherence core from the current master/slave topology; mutable
multi-master coordination remains future work.


### Shared snapshot cluster and automatic placement

The first cluster execution milestone supports unordered scalar reductions on
**immutable, fully rebuilt numeric shards** in a shared backend. Give each node
its own data/settings/system namespace, configure the same shared database
backend, and use the same secret file (at least 32 random bytes):

```sh
./memcp --no-repl --api-port=4321 \
  --cluster-secret-file=/run/memcp/cluster.secret \
  --cluster-databases=analytics \
  --cluster-address=http://node-a:4321/cluster lib/main.scm
```

Other nodes add `--cluster-join=http://node-a:4321/cluster` and advertise their
own reachable address. The contact is only a seed: every participant retains
the full peer roster and connects directly to discovered peers. The secret is
read in Go, never sent in the challenge proof or included as a CLI value.
Use HTTPS/WSS when the network needs transport integrity/confidentiality;
challenge authentication does not encrypt plain WebSocket traffic.

Ordinary SQL `SUM` now uses the existing shard-local scan pipeline. Before
loading a shard, placement checks warm peers and the executing node's RAM
budget; a typed acquisition panic delegates the reduction when appropriate.
The worker returns one scalar, never rows/shard dumps. `cluster_status` exposes
membership, executed/delegated counts and the tracked cache/admission budget.
`MaxRamBytes` bounds tracked cache plus numeric operator reservations; it is
not an RSS limit for the Go runtime, frontends or arbitrary user code.

The initial adapter mounts the named shared databases with S-only permissions.
Durable writes/DDL, rebuild and GC on those mounts are disabled/deferred until
cluster M handover is connected. Other namespaces retain normal behavior.
Automatic placement currently supports stored integer/float/sequence formats,
unfiltered scalar unordered CS-autocommit reductions; explicit transactions, grouped
results, query-local/physical sources and unsupported encodings fail explicitly.
This does not yet implement mutable multi-master databases or nested clusters.

The Python process runner has a focused `--cluster-only` scenario, run by the
**Cluster Stress test** CI job with normal and race-instrumented servers. Eight
shards exceed each worker's 8 MiB tracked budget; a 1 MiB coordinator
cannot load a shard locally. Three nodes calculate the exact SQL sum, exercise RAM and residency delegation, reject a wrong secret, release
all reservations and continue after losing the seed. Shared generation/schema
hashes must stay unchanged.

### Remote shard reductions

Start both participants with `--cluster-secret=<shared-secret>` (at least 16
bytes). `lib/main.scm` initializes `node_scan_start` and mounts the existing
`node_protocol` handler at `/cluster`. Custom Scheme servers can use the same
builtins. Replication authentication stays independent; replication peers are
never given this secret by the protocol.

The `scan` role mutually authenticates peers with fresh nonces and HMAC-SHA256
proofs bound to direction, protocol and process incarnation. The secret is never
sent to the connecting peer. Each node can execute scans or request work from
another node; `node_id` returns its public process incarnation.

For example, invoke through the ordinary authenticated Scheme API:

```scheme
(scan_remote nil "ws://worker:4321/cluster" "analytics" "events"
  '("amount") (lambda (amount) (> amount 100))
  '("amount") (lambda (sum amount) (+ sum amount))
  0 +)
```

This operator describes only the requested remote table, creates query-local
remote shard stubs, and delegates their reads over the shared binary WebSocket
protocol. **Both filter and mapreducer are serialized with Scheme `Serialize`,
including closed captures, and run in the worker's native shard scan pipeline.**
The coordinator receives one partial reduction per shard and applies `combine`.
It receives no rows, row callbacks or physical record-ID lists.

The optional final argument `"group_sums"` returns grouped reductions as a
FastDict. Use an association reducer, an empty association neutral and a combine
function such as `(lambda (a b) (merge_assoc a b +))`. The default `"scalar"`
accepts scalar reductions only. Each aggregate reply is bounded to 192 KiB;
oversized results fail explicitly.

The worker acquires schema/shard access through the existing MOSI infrastructure.
Tasks bind the physical generation, column-semantics fingerprint and a captured
main/delta record horizon. Later inserts are excluded. A changed generation or
schema aborts the task. Supported visibility is cursor stability for standalone
or autocommit reads; pass the current transaction as the first argument when
invoked within an executor. ACID/explicit transactions, mutation columns,
side-effecting callbacks and query-local handles are rejected. No task is
retried after an error or ambiguous completion. Task/attempt identities are
checked and replayed/expired task numbers are rejected per connection.

Explicit `scan_remote` remains available independently of membership. Shared
snapshot mounts use the automatic, budgeted path described above; mutable
distributed writes and general query placement remain subsequent features.

### Storage failure notifications

Administrators can register named Scheme callbacks for persistence failures.
The callback runs asynchronously, so notification delivery cannot delay or
rescue the failing storage operation:

```scheme
(register_storage_failure_hook "monitor" (list "*") 300 (lambda (failure)
	(http_request "POST" "https://monitor.example/memcp"
		(list "Content-Type" "text/plain") (serialize failure))))
```

The numeric argument is a per-fingerprint cooldown in seconds. A fingerprint
consists of the hook name, `class`, `backend`, `database`, and `operation`; the next delivered
event reports intervening occurrences in `suppressed_count`. Events also contain
`error`, `outcome_unknown`, and `timestamp`. Named hooks can be replaced or
removed independently. Callback panics and storage errors
caused by the callback are isolated and written directly to stderr instead of
being logged through a MemCP table.

The dashboard Settings page manages persistent definitions in
`system.storage_failure_hooks`. Each row stores its enabled state, selected
failure classes, cooldown, and Scheme source. Enabled definitions are compiled
once during startup and published to the in-memory dispatcher; failure handling
does not query the catalog. New installations include disabled templates for
calling the system logger and opening a local outage HTML page. Direct runtime
registrations are not persisted across process restarts.

### Test coverage

The repository currently contains more than 6,000 SQL/YAML cases across more
than 270 suites, in addition to Scheme runtime and Go storage tests. The SQL
taxonomy covers language semantics, planning, physical execution, concurrency,
storage, persistence, recovery, imports, RDF/SPARQL, anonymized integration
shapes, and performance regressions.

### Generated documentation

The generated API/reference documentation is not versioned. Build it locally when needed:

```bash
make docs
```

This writes the generated files to `docs/`.

## Remote storage backends

MemCP supports storing databases on remote storage backends instead of the local filesystem. To configure a remote backend, create a JSON configuration file in the data folder instead of a directory.

### S3 / MinIO Storage

Store your database on Amazon S3 or any S3-compatible storage (MinIO, Ceph RGW, etc.).

**Configuration file** (`data/mydb.json`):
```json
{
  "backend": "s3",
  "access_key_id": "your-access-key",
  "secret_access_key": "your-secret-key",
  "region": "us-east-1",
  "bucket": "memcp-data",
  "prefix": "databases"
}
```

**For MinIO or self-hosted S3-compatible storage:**
```json
{
  "backend": "s3",
  "access_key_id": "minioadmin",
  "secret_access_key": "minioadmin",
  "endpoint": "http://localhost:9000",
  "bucket": "memcp",
  "prefix": "data",
  "force_path_style": true
}
```

**Quick MinIO setup for testing:**
```bash
# Start MinIO with Docker
docker run -d --name minio \
  -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"

# Create a bucket (via MinIO Console at http://localhost:9001)
# Or via mc CLI:
mc alias set local http://localhost:9000 minioadmin minioadmin
mc mb local/memcp
```

### Ceph/RADOS Storage

Store your database directly on Ceph RADOS for high-performance distributed storage.

**Why is Ceph optional?** The Ceph backend uses CGO to link against `librados` (the Ceph client library). This requires the C headers and library to be installed at compile time and the shared library at runtime. To keep the default build simple and portable, Ceph support is behind a build tag.

```bash
# Install Ceph development libraries (Ubuntu/Debian)
sudo apt-get install librados-dev

# Build MemCP with Ceph support
make ceph
# or: go build -tags=ceph
```

**Configuration file** (`data/mydb.json`):
```json
{
  "backend": "ceph",
  "username": "client.memcp",
  "cluster": "ceph",
  "pool": "memcp",
  "prefix": "databases"
}
```

**Optional fields:**
- `conf_file`: Path to ceph.conf (defaults to `/etc/ceph/ceph.conf`)

**Setting up a Ceph development cluster with vstart.sh:**
```bash
# Clone Ceph source
git clone https://github.com/ceph/ceph.git
cd ceph

# Install dependencies and build (only vstart target needed)
./install-deps.sh
pip install cython setuptools
./do_cmake.sh
cd build && ninja vstart

# Start a development cluster
cd ..
MON=1 OSD=3 MDS=0 MGR=1 ./build/bin/vstart.sh -d -n -x

# Create a pool for MemCP
./build/bin/ceph osd pool create memcp 32

# Create a user for MemCP (optional, can also use client.admin)
./build/bin/ceph auth get-or-create client.memcp \
  mon 'allow r' \
  osd 'allow rwx pool=memcp' \
  -o ceph.client.memcp.keyring
```

**Environment variables for vstart cluster:**
```bash
export CEPH_CONF=/path/to/ceph/build/ceph.conf
export CEPH_KEYRING=/path/to/ceph/build/keyring
```

### Backend Configuration Reference

| Field | Backend | Description |
|-------|---------|-------------|
| `backend` | all | Backend type: `"s3"` or `"ceph"` |
| `prefix` | all | Object key prefix for database objects |
| `access_key_id` | S3 | AWS or S3-compatible access key |
| `secret_access_key` | S3 | AWS or S3-compatible secret key |
| `region` | S3 | AWS region (e.g., `"us-east-1"`) |
| `endpoint` | S3 | Custom endpoint URL (for MinIO, etc.) |
| `bucket` | S3 | S3 bucket name |
| `force_path_style` | S3 | Use path-style URLs (required for MinIO) |
| `username` | Ceph | Ceph user (e.g., `"client.admin"`) |
| `cluster` | Ceph | Cluster name (usually `"ceph"`) |
| `conf_file` | Ceph | Path to ceph.conf (optional) |
| `pool` | Ceph | RADOS pool name |

## Quick start

```bash
# 1. Build MemCP
go mod download
make

# 2. Start the HTTP and MySQL interfaces
./memcp --api-port=4321 --mysql-port=3307 lib/main.scm

# Run as a background daemon (use --no-repl to avoid exiting when stdin closes)
./memcp --no-repl --api-port=4321 --mysql-port=3307 lib/main.scm &

# 3. Create your first database
curl -X POST http://localhost:4321/sql/system \
  -d "CREATE DATABASE myapp" \
  -u root:admin

# 4. Create a table
curl -X POST http://localhost:4321/sql/myapp \
  -d "CREATE TABLE products (id INT, name VARCHAR(100), price DECIMAL(10,2))" \
  -u root:admin

```

### Optional PHP hosting

`make php` builds an application-independent PHP host with FrankenPHP and the
additional `memcp:` PDO driver. See [Embedded PHP](README-PHP.md) for external
PHP-ZTS dependencies, configuration, driver coexistence, and tests.

### Build with the experimental JIT

MemCP can compile Scheme expressions and query-planning code to native code. This
integration needs the `runtime/jit` support from the
[launix-de Go fork](https://github.com/launix-de/go/tree/jit-foreign-frames-go1.27.0):

```bash
# Build the patched Go toolchain next to the MemCP checkout.
git clone --branch jit-foreign-frames-go1.27.0 \
  https://github.com/launix-de/go.git ../go-jit
(cd ../go-jit/src && ./make.bash)

# Build MemCP with JIT support enabled.
PATH="$(cd ../go-jit/bin && pwd):$PATH" GOEXPERIMENT=jit go build -o memcp

# The experiment must also be enabled when running Go tests.
PATH="$(cd ../go-jit/bin && pwd):$PATH" GOEXPERIMENT=jit go test ./scm
```

The native JIT backends support AMD64, ARM64 and RISC-V64. `jitgen` runs on the
host and emits one shared set of Go emitters, not one set per architecture.
Algorithms use semantic operations and register roles from the shared API; instruction selection,
register assignment and atomic memory ordering belong to the target backend.
Type sizes follow the loaded package's target architecture. All three backends
use the same 64-bit value layout; 32-bit targets are not supported by the JIT.
`make jitgen` builds the generator for the host even when `GOARCH` selects a
cross-compilation target. Regeneration is checked in JIT CI.

For Linux cross-tests, install `qemu-user-static` and use the patched toolchain:

```bash
make jit-cross-test JIT_CROSS_ARCH=arm64
make jit-cross-test JIT_CROSS_ARCH=riscv64
```

These targets cross-compile and execute the Scheme and storage Go tests under
QEMU with PHP disabled. They require no target C compiler (`CGO_ENABLED=0`).
The JIT CI runs the same architecture matrix.

The JIT is deliberately guarded by `GOEXPERIMENT=jit`. A normal build with an
official, unpatched Go compiler remains supported and uses the interpreter; it
does not compile or activate the experimental runtime integration.

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--api-port=PORT` | `4321` | HTTP API listen port |
| `--mysql-port=PORT` | `3307` | MySQL protocol listen port |
| `--mysql-socket=PATH` | `/tmp/memcp.sock` | MySQL Unix socket path |
| `--root-password=PASSWORD` | `admin` | Initial root password (first run only) |
| `--root-password-file=PATH` | — | Read the initial root password from a file |
| `--disable-api` | — | Disable HTTP API server |
| `--disable-mysql` | — | Disable MySQL protocol server |
| `--no-repl` | — | Disable interactive REPL (required for daemon/background use) |
| `--initialize` | — | Initialize a fresh data directory and exit cleanly |
| `-data DIR` | `./data` | Data directory |

### Authentication

> **Security note:** Never expose MemCP directly to the internet with default credentials. Always set a strong `--root-password` before any network-accessible deployment.

- Source builds default to `root` / `admin` unless `--root-password` is supplied
  for the first start of a fresh data directory.
- DEB and RPM installations generate a random initial password. Read it once
  with `sudo cat /etc/memcp/initial-root-password`, change it, and then delete
  that file.
- Containers require either the `memcp_root_password` secret shown below or an
  explicit `ROOT_PASSWORD` for a fresh volume. The secret is preferred because
  environment variables are visible through container metadata.
- Change the credentials with:
```bash
curl -X POST http://localhost:4321/sql/system \
  -d "ALTER USER root IDENTIFIED BY 'new-long-random-password'" \
  -u root:"$OLD_PASSWORD"
```

## Installation packages

Tagged releases publish a static Linux binary, an amd64 DEB, an x86_64 RPM and
source RPM, checksums, and a multi-architecture container image for amd64 and
arm64. The version in the tag must exactly match the first word in
`CHANGELOG.md` (for example `v0.2` for version `0.2`).

`make`, `make jit`, and the DEB/RPM packages include embedded PHP. Packages
supply their matching ZTS runtime, PHP configuration, and Imagick module.
The standalone static release binary uses the explicit `nophp` build.

### Debian and Ubuntu

```bash
sudo apt install ./memcp_VERSION_amd64.deb
sudo cat /etc/memcp/initial-root-password
systemctl status memcp
```

The password file is readable only by root. After logging in and changing the
password, remove the one-time copy:

```bash
sudo rm /etc/memcp/initial-root-password
```

Configuration lives in `/etc/memcp/memcp.conf`, persistent data in
`/var/lib/memcp`, and the local MySQL socket in `/run/memcp/memcp.sock`.
Package upgrades stop MemCP gracefully before replacing the binary and restart
it afterwards. Removing or purging the package deliberately preserves database
data, the service account, and any still-existing initial password file.

The packaged systemd sandbox only permits writes below `/var/lib/memcp` and
`/run/memcp`. If `-data` is moved elsewhere, add that absolute directory with a
systemd drop-in (`ReadWritePaths=`); paths below `/home` additionally require an
appropriate `ProtectHome=` override.

### RPM distributions

```bash
sudo dnf install ./memcp_VERSION_x86_64.rpm
sudo cat /etc/memcp/initial-root-password
systemctl status memcp
```

RPM installations use the same paths and data-retention rules as DEB packages.

### Container deployment

Create `memcp-root-password.txt` outside the repository and keep it out of Git:

```yaml
services:
  memcp:
    image: carli2/memcp:VERSION
    ports:
      - "4321:4321"
      - "3307:3307"
    volumes:
      - memcp_data:/data
    secrets:
      - memcp_root_password

secrets:
  memcp_root_password:
    file: ./memcp-root-password.txt

volumes:
  memcp_data: {}
```

The image runs as the fixed unprivileged user/group `10001:10001`. Existing
host directories mounted at `/data` must therefore be writable by UID 10001.
The initial password is read only when `/data` is fresh; subsequent starts use
the credentials stored in the database.

For compatibility, `ROOT_PASSWORD` is also accepted on the first start:

```yaml
services:
  memcp:
    image: carli2/memcp:VERSION
    environment:
      ROOT_PASSWORD: replace-with-a-long-random-password
    ports:
      - "4321:4321"
      - "3307:3307"
    volumes:
      - memcp_data:/data
volumes:
  memcp_data: {}
```

### Building packages locally

```bash
make memcp.deb
make memcp.rpm
python3 tools/test_packaging.py --artifacts
```

Artifacts are written to `dist/`. Container builds are not part of these local
targets. `make docker-release` is an explicit multi-architecture push and
requires an authenticated Docker Buildx installation.

### Publishing a release

Repository maintainers configure `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` as
GitHub Actions secrets. After the release changelog PR is merged, create and
push an annotated tag whose name exactly matches `v` plus the changelog version.
The release workflow verifies that the commit belongs to `master`, rebuilds and
checks all artifacts, publishes the versioned and `latest` container tags,
creates SHA256 checksums and build-provenance attestations, and only then creates
the GitHub release. It must not be replaced by manually uploading locally built
files.

## Importing existing data

MemCP can bulk-import schema and data from MySQL or PostgreSQL with a single
Scheme call. The import **drops and recreates the selected target tables** on
every run. This makes an import reproducible after schema changes, but it is a
destructive replacement of existing target data. Use a separate target database
or take a verified backup when that data must be retained.

### Import from MySQL

```scheme
; import all databases (skip system dbs)
(mysql_import nil nil "root" "secret")

; import one specific database
(mysql_import nil nil "root" "secret" "myapp")

; import into a differently-named MemCP database
(mysql_import nil nil "root" "secret" "myapp" "myapp_memcp")
```

Parameters: `host` (nil → 127.0.0.1), `port` (nil → 3306), `username`, `password`,
`sourcedb` (nil → all), `targetdb` (nil → sourcedb), `sourcetable` (nil → all), `targettable` (nil → sourcetable).

`mysql_import` connects to a running MySQL/MariaDB server over the wire
protocol. Row copying is parallelised across up to 8 workers, and trigger
metadata in `system.triggers` is reset at the start of each run.

#### Alternative: replay a `mysqldump` file

When you only have a dump file and no live server, replay it through any MySQL
client against MemCP's own wire-protocol port. MemCP's SQL parser accepts the
constructs `mysqldump` emits — versioned `/*!… */` executable comments
(including split `CREATE`/`DEFINER`/`TRIGGER` blocks), `LOCK TABLES` /
`UNLOCK TABLES`, savepoint and session-isolation statements, and the
`INFORMATION_SCHEMA.FILES` tablespace probes:

```bash
mysql -h 127.0.0.1 -P 3307 -u root -p myapp < dump.sql
```

Take the dump with `--databases` (or prepend `CREATE DATABASE` / `USE`) so the
statements target the intended database.

### Import from PostgreSQL

PostgreSQL has an extra hierarchy level — **database → schema → table** — compared to MySQL.
The `sourceschema` parameter selects which schema(s) within the database to import.
All imported schemas land in the same MemCP database (`targetdb`).

```scheme
; import the public schema of one PostgreSQL database
(psql_import nil nil "postgres" "secret" "myapp" "public")

; import all non-system schemas of one database
(psql_import nil nil "postgres" "secret" "myapp" nil "myapp")

; import all databases (each becomes a separate MemCP database)
(psql_import nil nil "postgres" "secret")
```

Parameters: `host` (nil → 127.0.0.1), `port` (nil → 5432), `username`, `password`,
`sourcedb` (nil → all), `sourceschema` (nil → all non-system schemas), `targetdb` (nil → sourcedb),
`sourcetable` (nil → all), `targettable` (nil → sourcetable).

Both functions print a line for each imported table and return `true` on success.

#### Alternative: load a `pg_dump` file

MemCP has no PostgreSQL wire protocol, so a dump file is replayed by the
built-in dump loader rather than by a client. `load_psql` takes the target
MemCP database, the dump source, and a policy:

```scheme
(load_psql "myapp" "/path/to/dump.sql" (sql_policy "root"))
(load_psql "myapp" "/path/to/dump.sql.gz" (sql_policy "root"))
(load_psql "myapp" "/path/to/dump.tar" (sql_policy "root"))
```

It accepts a plain-format dump (`pg_dump -Fp`, the default) or a tar-format
archive (`pg_dump -Ft`), each optionally gzip-compressed; the loader reads
`COPY … FROM '…'` data files out of the archive automatically. The custom
format (`-Fc`) is not supported. The loader skips `psql` meta-commands and
`CREATE FUNCTION` bodies, applies `CREATE TABLE` / `ALTER TABLE`, streams both
`COPY … FROM stdin` and `COPY … FROM 'file'` data, retargets the dump's
database/schema name onto the target database, and replays
`pg_catalog.setval()` so auto-increment sequences resume correctly. Run it from
the REPL or the `/scm` endpoint. For large dumps, use the REPL: HTTP requests
have a five-minute write timeout, so `/scm` can lose its connection before a
long import finishes. Check table counts before retrying an interrupted request.

## Use cases

MemCP is designed for applications that mix ordinary reads and writes with
large or structurally complex reads, for example:

- **drop-in replacement for an existing database once it holds too much data
  and has become slow**, keeping the same SQL application in place;
- dashboards, reports, and grouped statistics;
- ordered and paged lists over large datasets;
- full-text-like searches combined with relational filters;
- multi-tenant or row-level ACL checks expressed through joins and correlated
  subqueries;
- notification and maintenance queries that repeatedly reuse the same grouped
  facts;
- RDF/SPARQL datasets alongside relational application data;
- development and compatibility testing through MySQL clients or SQL-over-HTTP.

## Contributing

Contributions should include focused regression coverage and preserve the
logical/physical planner boundary documented in `INVARIANTS.md`.

### CI behavior

- Pull requests to `master` run the full required `test` workflow.
- Direct pushes run `test` only on `master` (to avoid duplicate branch + PR runs).
- If a PR shows pending/old checks, push one fresh commit so the current workflow config is applied.

Useful contributions include SQL compatibility cases, correctness fixes,
measured performance work, documentation, storage formats, and new operators.

### Getting started

Fork the repository, then build and test your clone as described under
[Quick start](#quick-start) and [Testing](#testing). Develop on a branch, add
regression coverage for every change, and open a pull request against `master`.
Use a dedicated worktree under `.worktrees/` and fetch/rebase onto the current
`origin/master` before every push. See [Repository Guidelines](AGENTS.md) for
the complete workflow.

## Testing

Run focused local tests for the feature or bug being changed. Intermediate
commits may contain failing or incomplete work; document known failures and use
a draft PR when appropriate. GitHub CI runs the full suite for code changes,
and required checks must pass before merging into `master`. Documentation-only
changes retain the existing CI skip policy.

`git-pre-commit` remains the CI full-suite runner invoked by `make test`; do not
install it as a local commit hook or run full tests during local patch development.
If the legacy full-suite hook is already installed, locate it with
`git rev-parse --git-path hooks/pre-commit`, inspect it, and move that legacy hook
to a backup name. Preserve unrelated hooks. Ordinary commits do not need
`--no-verify`.

```bash
# Run a focused test suite (starts its own server automatically)
python3 run_sql_tests.py tests/sql/expressions/basic-sql.yaml   # Basic operations
python3 run_sql_tests.py tests/sql/expressions/functions.yaml   # SQL functions
python3 run_sql_tests.py tests/sql/expressions/error-cases.yaml # Error handling

# Connect to an already-running instance (skip startup)
python3 run_sql_tests.py tests/sql/expressions/basic-sql.yaml 4321 --connect-only
```

## License

MemCP is free software licensed under GPL-3.0-or-later. See [LICENSE](LICENSE).

Questions and design discussions are welcome in [GitHub
Discussions](https://github.com/launix-de/memcp/discussions).
