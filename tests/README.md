<!-- Copyright (C) 2026 Carl-Philip Hänsch -->

# SQL Test Taxonomy

SQL integration suites use descriptive lower kebab-case filenames without
sequence numbers. The directory is the stable ownership and selection unit:

- `sql/`: language semantics, compatibility, DDL, DML, security, and triggers
- `planner/`: logical and physical planning for aggregates, joins, ordering,
  set operations, and subqueries
- `execution/`: physical operators and concurrency behavior
- `storage/`: formats, indexes, persistence, and recovery
- `integration/`: imports, anonymized application regressions, and composed
  read-model/query shapes
- `rdf/`: RDF, SPARQL, Turtle, and RDFHP behavior
- `performance/`: explicit latency, scaling, and benchmark suites

Every YAML suite is discovered recursively by `git-pre-commit`. A suite may
opt out only with `metadata.ci: false`, which is reserved for manual benchmarks
that do not assert a stable CI budget. Every suite must provide
`metadata.description`.

`metadata.cluster: true` selects the cluster process runner instead of the normal
SQL runner. Changing that one parameter is sufficient to move an ordinary suite:
`setup`, `test_cases`, SQL/SCM, sessions, repeats, expectations and `cleanup` retain
their normal format and executor. The normal runner reports an explicit skip
before opening a connection or starting a server. This is an execution profile,
not `ci: false`: cluster CI discovers and runs these suites separately.

The marker may also be a mapping, for example:

```yaml
metadata:
  description: Distributed ordered scan
  cluster:
    nodes: 4
    ram_limit_bytes: 8388608
    coordinator_ram_limit_bytes: 131072
    shard_ram_percent: 50
    coordinator_shard_ram_percent: 1
    gomaxprocs: 16
test_cases:
  - name: Ordered result requires remote shard execution
    sql: SELECT id FROM items ORDER BY id LIMIT 100
    expect:
      rows: 100
      cluster:
        min_remote_shards: 1
        min_remote_nodes: 2
        data_exceeds_node_ram: true
        plan_contains: [scan_order]
```

These are the defaults for `cluster: true`; nodes may range from 2 to 10. RAM
limits are native `MaxRamBytes`/`MaxPersistBytes` cache budgets in bytes, not OS
process-RSS limits. Fractions are 1–100 percent, and concurrency is node-local
GOMAXPROCS. Unknown properties and assertions fail instead of being ignored.
The shard fraction reserves native payload RAM independently of caches and
indexes. Each table may use one third of that shard budget, leaving room for
secondary and tertiary scans on the same worker. These are admission thresholds;
active readers retain their payload until release, when reclamation retries.
Modified transaction generations retain their native pins and may use the
remaining node-wide shard budget beyond that per-table read-working-set share.
`compare_standalone` defaults to true and runs the same YAML on an independent
standalone oracle first, comparing complete case results as well as ordinary
expectations. Disable that comparison for fixtures with intentionally different
valid results across independent runs. Remote execution remains mandatory.
For a query without an ordering contract, `compare_reference_order: false` on
the testcase compares the complete row multiset, preserving duplicate counts.
The default retains exact row order. Shared analytics runs start the oracle and
cluster from separate copies of the same committed fixture, so oracle mutations
cannot change the cluster's starting data.

Every cluster suite must demonstrate at least one delegated shard and one remote
worker through changes in native execution counters. Membership alone, a correct
SQL response or merely configuring several nodes does not pass. Individual cases
may require more shards or distinct workers with `expect.cluster`. Counters are
sampled around the testcase operation, excluding setup, cleanup and EXPLAIN.
Missing measurements, reset counters
and repeated node identities fail. Per-case counters cannot attribute overlapping
parallel-group requests: explicit per-case cluster assertions there fail clearly;
ordinary parallel groups retain their concurrency. Their overlapping node counters
do not constitute per-query evidence, so a parallel-only suite without an
attributable remote operation fails rather than borrowing another case's work.
Restart and atomic-schema observation actions need a dedicated recovery scenario;
they fail explicitly instead of being treated as a single-server restart.

`data_exceeds_node_ram` compares measured active persistent generation file bytes
with the largest *actual* node RAM budget. WAL, schema and retired files do not
inflate that proof. Setup must persist the fixture when requesting it. This is
evidence of a dataset larger than one node's budget; it is not a measurement of
the query's resident working set or proof that streaming on one node is impossible.
Use the execution assertions independently to require real distribution.

Cluster setup normally runs against an initially empty shared `memcp-tests`
database. The optional `fixture: shared_analytics` preset retains the existing
large analytic fixture; its `rows_per_shard`, `shards`, `columns` and `fixture_seed`
are declared in the operator suite. Its generation algorithm and process/crash
orchestration remain in `tools/`; SQL operator case definitions live under `tests/`.

```sh
python3 tools/replication_cluster_stress.py --cluster-suites
python3 tools/replication_cluster_stress.py tests/execution/operators/cluster-operator-contracts.yaml
```

A case with `noncritical: true` still runs and reports failures, but its failures
do not block CI. The T-SQL compatibility suites retain the complete behavior
inventory, including currently unsupported features. Remove the flag after the
case passes focused local verification; keep its assertions unchanged. Expected
error cases specify rejection behavior and do not establish support for the
corresponding successful operation.

In these suites, fixture preparation and cleanup are explicit cases so missing
DDL remains visible without preventing the remaining specifications from running.
All suites remain enabled in CI and use exclusive scheduling. A suite with
`metadata.requires_tds: true` starts the native listener at HTTP port + 2000.
Its `tds` cases run `tools/tdsprobe` against that listener using a real client
driver. Each probe keeps its default connection across statements; an optional
statement `connection` label selects an independent connection. `reset: true`
requests the driver's native session reset before SQL, and `close: true` without
SQL closes that label's socket. Probes support named parameters and prepared
calls, and check rows, values, column names and errors. `expect.messages` checks
real TDS INFO tokens in order; `messages: []` asserts that none were received.
Message assertions may specify number, state, severity, text, server, procedure
and line independently. They remain active when an error is also expected, so
completed rows and informational messages before a later failure are verified.
`sql/compatibility/tsql-print-messages.yaml` covers PRINT, informational RAISERROR,
multiple resultsets, transaction state and database ENVCHANGE on error. HTTP
streams carry notices as comments rather than relational result rows.
Unsupported frontend features retain explicit rejection tests; implemented
regressions are critical.

`sql/compatibility/tsql-server-properties.yaml` covers the isolated server-property
profile: `Edition` identifies MemCP; `InstanceDefaultDataPath` and
`InstanceDefaultLogPath` report the virtual Windows labels `C:\MemCP\Data\` and
`C:\MemCP\Log\`. These labels never map to host files or disclose physical
locations, even for administrators. Client-supplied filesystem operations remain
unsupported and must not resolve these labels into host paths. Other
properties use the documented bounded virtual server profile: the compatibility
version matches the advertised TDS login version; host identity and unknown
physical resources remain NULL. This profile does not promise support for
unimplemented SQL features. Property expressions are resolved at execution time,
and TDS declares SQL_VARIANT even for NULL results. Each property retains its
declared intrinsic subtype, including NVARCHAR and INT.
HTTP output unwraps that scalar at the result edge; explicit text casts are
supported. General SQL_VARIANT column storage, implicit conversions, comparisons,
and non-text subtype conversions are outside this property profile.


`sql/compatibility/tsql-database-options.yaml` covers native database creation
with virtual data/log file labels. Every label is validated before creation;
SIZE remains a compatibility hint and does not reserve host disk space. The
bounded option profile accepts RECOVERY SIMPLE, AUTO_CLOSE OFF and AUTO_SHRINK
OFF. TRUSTWORTHY ON/OFF is persisted frontend catalog state only: no native
permission, impersonation, registry, process or host operation consults it.
Unsupported recovery and resource modes fail explicitly. Database DROP/recreation
retires the previous frontend state.

`sql/compatibility/tsql-row-guid-declarations.yaml` covers CREATE TABLE
ROWGUIDCOL as immutable column declaration metadata, restricted to one direct
UNIQUEIDENTIFIER column per table. It introduces no default, uniqueness or
nullability constraint. Native `show` exposes generic scalar `Properties`; the
T-SQL metadata wrapper derives `IsRowGuidCol`. Marked ALTER ADD, `$ROWGUID` and
COLUMNPROPERTY remain explicitly unsupported pending atomic declaration or
logical binding support. Permanent CREATE inside an explicit transaction is
rejected natively until transactional catalog publication exists.

`sql/compatibility/tsql-virtual-filesystem.yaml` covers the frontend-owned
virtual filesystem interface. `xp_fileexist` and `SERVERPROPERTY` share the
same logical roots (`C:\MemCP\Data\`, `C:\MemCP\Log\`) and ancestors.
The interface performs no host file, network, shell or backend-directory I/O.
Only declared virtual directories exist; no physical database files or free
space are fabricated. Unknown virtual entries report absent. Windows case and
separators normalize consistently; UNC/device paths, unsupported drives,
drive-relative paths, escaping traversal, alternate streams, control bytes,
wildcards and ambiguous/reserved Windows names are rejected. URL escapes are
not decoded. SQL administrator privileges never authorize host filesystem
access. `xp_fileexist` supports the ordinary three INT result columns, typed
input variables and an optional declared OUTPUT variable for file existence.
Directory listing, free-space reporting, file writes/deletion, executable
loading and shell execution remain unsupported rather than reporting success.


Place a regression at the layer that owns its root cause, not at the layer
where an application happened to expose it. Use `integration/query-shapes`
only when the composition of otherwise independent SQL features is the subject
of the test. Keep one topic per suite; split a file when setup and assertions
cover independent planner or execution contracts.

Run one suite directly with, for example:

```sh
python3 run_sql_tests.py tests/planner/subqueries/deep-correlation-membership.yaml
```

Replication process/crash scenarios run outside the Go test harness. Build
`memcp`, install the `mysql` client, then run:

```sh
python3 tools/replication_cluster_stress.py --seed 20261023 --rounds 5 --workers 4 --operations 25
```

The separate **Cluster SQL operator contracts** CI job compares ordered scans,
Top-N/OFFSET, computed and NULL sort keys, EXISTS, scalar lookups, joins and
membership scans against a standalone SQL oracle. The YAML operator suite includes
an unlimited ordered result spanning multiple pages, a long OFFSET/LIMIT window
crossing multiple ACKs, many streams with a small LIMIT and LIMIT 1 without ORDER BY.
Run it with:

```sh
python3 tools/replication_cluster_stress.py --cluster-operators-only
```

CI partitions the complete discovered suites across two jobs; each job runs
both normal and race-instrumented servers with the original standalone oracle,
fixture and distributed-execution assertions. The required aggregate check
passes only when both jobs succeed. Local `--cluster-suites` still runs every
suite; reproduce one CI partition with:

```sh
python3 tools/replication_cluster_stress.py --cluster-suites --suite-shard-index 0 --suite-shard-count 2
```

The artifact manifest records the complete discovered suite list, selected
suite paths and zero-based partition index/count. Empty or invalid partitions
fail before starting processes.

Each cluster query must increase the coordinator's remote-execution counter;
correct local results alone do not pass. Reports retain the SQL, EXPLAIN, oracle
rows, remote shard count and every failure. Unsupported operators fail CI;
there are no expected-failure exemptions. Native batch/lookup wire tests cover
physical operator contracts separately, since a SQL join may lower to another
operator such as `scan_join_order`.

The **Replication Cluster Stress test** workflow executes this runner directly.
The default run connects through `/cluster` on the normal HTTP API listener; the
race-instrumented process run uses `--dedicated-replication-port` to cover the
optional dedicated listener with the same binary WebSocket protocol.
It repeats the process scenarios with race-instrumented servers and a second
seed, failing on a Go race report and retaining both runs' artifacts.
It owns the server processes, alternates SIGKILL/restart without taking an owner
and its backup offline together, and runs concurrent slave writers, master
analytics and garbage collection. A separate scenario uses two simultaneous
slaves with different databases and a third master-only database. The latter
executes ADD/DROP COLUMN and exact data queries on its own master session,
including complete cycles while either slave is unavailable; neither slave may
acquire its catalog or ownership records.

The per-row oracle compares complete current values on both nodes at mutation
barriers and after recovery. A write interrupted before its response may have
either its previous or new value and is never blindly retried. Both nodes must
agree on that outcome; recovery fixes it permanently. Reclaim checks the schema
and every current shard, including empty, partitioned and internal durable
shards, before cutting the master connection and performing offline DDL/DML.
An unreachable holder must fail without ownership promotion. Logs, write events,
seed and a result manifest remain in the
printed artifact directory. `--executor` and `--startup-timeout` also support
running a prebuilt target binary through QEMU. Go tests retain only in-process
MOSI/state-machine and transport checks.

Run a taxonomy section through the pre-commit selector:

```sh
./git-pre-commit 'tests/planner/subqueries/*.yaml'
```

## Result validation and completion counts

Requested fields in `expect.data` must actually exist in each result object.
An explicit SQL NULL is represented by a present field with a null value;
`{}` does not satisfy an expectation for `{"value": null}`.

`expect.affected_rows` validates one numeric HTTP completion object. The
legacy empty-body exception applies only to an expected zero completion; it is
not evidence of successful schema installation. Keep schema and data readback
assertions alongside the completion check.

Ordinary MySQL CREATE/DROP TABLE and CREATE/DROP TRIGGER report zero affected
rows, including successful schema changes. Both REST and the MySQL wire execute
the same dialect-owned completion formula. Native Scheme DDL boolean results
remain separate from this SQL protocol contract. See the official
[MySQL ROW_COUNT documentation](https://dev.mysql.com/doc/refman/8.0/en/information-functions.html#function_row-count).
CREATE TABLE AS and ALTER TABLE have separate row-count rules and are not
covered by this ordinary declaration rule.

MySQL probe statements can specify `expect.affected_rows`; the probe compares
the driver's actual `RowsAffected()` result. It does not infer a count from an
empty result set. PostgreSQL command tags and TDS count-validity flags have their
own dialect contracts; this MySQL rule does not define their completion values.

Ordinary PostgreSQL CREATE/DROP TABLE completes with `command_tag` metadata and
no `affected_rows` field, including successful conditional reuse/removal. This
preserves the distinction between an absent row count and a valid zero. See the
[PostgreSQL libpq completion contract](https://www.postgresql.org/docs/current/libpq-exec.html#LIBPQ-PQCMDTUPLES)
and [CommandComplete protocol](https://www.postgresql.org/docs/current/protocol-message-formats.html).
The HTTP adapter renders that same frontend completion as an object such as
`{"command_tag":"CREATE TABLE"}`. There is currently no native PostgreSQL wire
server. PostgreSQL syntax tunneled through the existing MySQL connection is
still MySQL transport: its required OK-packet count is mapped to zero only in
that access adapter. It is not a PostgreSQL affected-row count. Numeric DML
completions remain unchanged.

T-SQL CREATE/DROP TABLE, CREATE/ALTER/DROP VIEW and source module declaration
commands complete without an affected-row count. Their HTTP response omits
`affected_rows`; the TDS completion clears `DONE_COUNT` and uses an unsigned
zero placeholder. A genuine zero-row UPDATE/DELETE still has a present zero
count. SET NOCOUNT additionally suppresses DML count publication, without
changing the native write or its internal count. See Microsoft's
[DONE validity flag](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tds/3c06f110-98bd-4d5b-b836-b1ba66452cb7)
and [ExecuteNonQuery contract](https://learn.microsoft.com/en-us/dotnet/api/system.data.sqlclient.sqlcommand.executenonquery).
The client's -1 sentinel for missing counts is never placed in the unsigned
TDS count field. Raw TDS probe packets can assert `done_count` (the final
DONE_COUNT bit) and `done_rows` (its unsigned count slot) separately.
The released go-mssqldb v0.12.3 driver's `ColumnTypePrecisionScale` returns
unknown for DATETIME2, TIME and DATETIMEOFFSET. Temporal scale assertions
therefore use raw `contains_hex` TYPEINFO fixtures, including the column name
immediately after the type/scale bytes; ordinary client probes retain result
names/types, nullability and values. Decimal/numeric `precision` and `scale`
assertions remain strict and never infer missing metadata from expectations.

This bounded declaration contract does not yet classify ALTER TABLE/INDEX or
transaction-control completions, and does not implement the complete sequence
of intermediate DONE/DONEINPROC/DONEPROC counts for mixed batches and stored
procedure execution. Existing statement execution and result sets remain shared
between HTTP and TDS; this is not a complete MSSQL completion implementation.

## Performance trade-offs between startup and repeated execution

When a change trades cold-start cost against warm execution, gate the complete
workload: one cold query plus a fixed number `n` of subsequent executions. The
cold request includes SQL compilation and lazy query/cache preparation; fixture
loading and server process startup remain outside the query measurement.

Use `timing_aggregation: total`, `warmup: 0`, and `timing_samples: n + 1`.
For a fixture that already separates cold and warm cases, give both the same
`timing_group` name, measure the cold case once and the warm case `n` times.
The `--perf-ab` runner compares the sum for that group, retaining each member's timing in the
artifacts. There are no discarded warmups or adaptive repetition counts.

Choose `n` before measuring, apply it identically to both revisions, and retain
it across verification trials. The initial trade-off fixtures use `n = 10`.

Short workloads also need enough measured work to resist fixed scheduling and
collection noise. The front-cache cases in `in-list-template.yaml` sum 51
requests, each creating a fresh cache and performing 40 lookups including cold
compilation. Every result is checked; no sample is discarded. The A/B percentage
gate and absolute budget stay unchanged. Its forced-compilation case already
does enough work with five requests and keeps that configuration.
The total may be at most 20% slower; no warmup bonus or fixed jitter allowance
extends that limit. Suspect totals receive the usual complete isolated-fixture
old/new verification, with medians taken across whole workload totals. Independent
latency tests keep their existing median-per-request policy.

Short fixed-total cases must contain enough repeated work that one scheduling
pause or garbage collection does not dominate the total. Reuse the fixture;
include the first execution and validate every response. Select the count from
baseline timings before A/B measurement, then freeze the same count for both
revisions and all verification pairs. Do not force collection, discard slow
samples, or retry until a green result appears.

The scan-kernel suite measures 1001 SQL requests and 101 direct SCM scans per
case. Typed-filter totals use 101 scans/transactional requests and 1001 cached
SQL requests. Literal dead-branch coverage measures 51 requests; cached
range-first cases measure 501. The two longer range-first mapping cases retain
11 requests. These counts amortize fixed disturbances without repeating the
expensive table setup. Existing one-shot cache-fill tests remain separate;
replacing their work with cache hits would change what they protect. Every
existing result assertion, absolute budget, and 20% A/B gate remains in force.

Ordered Unicode and compound-index measurements retain their original cold
request plus ten reuses as a named phase and add 490 further requests against
the same fixture/query. Each pair shares `timing_group`: the 20% gate compares
all 501 requests, while both raw phases remain visible. Their absolute budgets
are split between the phases, preserving the previous combined limit. No setup
or cold compilation is discarded, and every response is validated.

The FK-partition count and selective unique-point probes follow the same rule:
their original first eleven/four requests remain visible, followed by enough
identical requests for 1001/1000 measured calls. The combined absolute budgets
and 20% group gates remain unchanged; no collection or first query is removed.

- `sql/dml/insert-values-template.yaml`: Bulk INSERT literals, session bindings and computed cells.
- `performance/bulk-insert-compile.yaml`: Cold compilation of parameterized bulk INSERT rows.
- `performance/jit-scan-kernel-compile.yaml`: Full column scans including per-invocation filter and reducer compilation; one cold request plus 1000 SQL or 100 SCM repetitions.
- `performance/joined-aggregate-driver.yaml`: Selective keyed reductions and dense
  unique/nonunique joins over 360,000 rows at the default shard size. Each new
  sparse reduction totals 101 requests; each slower dense control totals 11.
  All include cold compilation and lazy preparation. The same cases feed
  `go run ./tools/costgen --validate-only
  --suite joined-aggregate-driver`, which checks forced-plan results and the
  measured ranking of nested scans against `scan_join_order`.

Compile-time guards execute `EXPLAIN COMPILE` eleven times against the same
fixture. Each request builds a complete new plan; these are not reads of an
execution-plan cache. Planner and phase timing limits apply to the arithmetic
mean of all eleven measurements, including the first and any collection pause.
Structural metrics use the maximum, so no oversized plan can disappear in an
average. A compilation error or missing requested metric fails the test.
`--log-times` retains every raw metrics object as `COMPILE_TIME` output. Machine
scaling and all existing per-compilation budgets remain unchanged.

## Diagnose a red timing check

The SQL runner validates the returned result before applying execution-time
budgets. A timing failure reports `result assertions passed`; an expectation
mismatch reports `result correctness` even when the query was also slow. A
repeated SELECT/SCM measurement checks every returned result, including warmup
responses, in standalone and A/B runs. Validation occurs outside the timed
request; the first mismatch stops the measurement before a later response can
hide it. Existing interrupted-response and baseline-recovery rules remain active.
An ordinary planner-time or plan-size failure occurs before query execution and reports that
the result has not yet been checked. Passing standard SQL queries also print their
measured latency and applicable hard budget, so CI logs preserve the margin as
well as failures. These distinctions do not waive any gate.

An absolute `max_time` failure alone does not establish a regression relative to
master. Compare the exact baseline revision and candidate on identical fixtures,
including cold/warm phases, and retain all fixed verification trials. Check the
baseline CI log too: a baseline which exceeds the same absolute budget is evidence
of an existing budget problem, not proof that the candidate is harmless. Query
errors, wrong results and timeouts still require investigation.

Absolute limits are runaway backstops; the paired performance A/B job detects
relative slowdowns. The pre-server SHA-256 machine calibration is independent of
MemCP, but it is not a calibration of allocation, garbage collection or memory
access costs. Before proposing a budget-policy change, collect repeated baseline
measurements and describe the margin, measurement isolation and intended failure
mode. Changes to protected calibration or policy code require the maintainer's
policy-update procedure; do not hide a budget relaxation in an engine patch.

## Valid SQL with missing implementation

A missing implementation is a positive behavior test marked `noncritical: true`,
not an assertion that valid SQL must fail. Promote it to critical after the
implementation and its result are proved. The historical case identity remains.

Only the exact reviewed [implementation-gap inventory](implementation-gaps.md)
may migrate an old error assertion or become noncritical. Its protected manifest
binds SQL, dialect, fixture prefix and the full approved result oracle. Regular
syntax/type/ACL/security errors and all timing/plan gates remain protected.
Policy changes require the separate administrative PR/review procedure described
there; preparing the PR grants no merge or bypass authority.

## Baseline queries that cannot execute

In `--perf-ab`, a baseline query error or request timeout is recorded as
`baseline_failure`, with its phase, timeout and diagnostic. It is not a timing
sample and does not imply a speedup. Only the orchestrator's A role can produce
this outcome. Setup/cleanup failures, connection failures, malformed artifacts,
wrong results and all candidate errors still fail the run.

Such cases receive the same fixed verification as suspected regressions: the
initial A/B pair plus six additional isolated fixtures per revision (six
old/new pairs). Every candidate sample must satisfy the result assertions. Every
candidate fixture must also meet the declared `threshold_ms` budget; timing
groups use the sum of member budgets and measured totals, preserving cold/warm
trade-offs. Regression waivers cannot bypass this recovery budget.

If any baseline fixture succeeds, all successful baseline timings remain in the
normal regression comparison. If all seven baseline fixtures fail and the
candidate passes every fixture, the report says `NEWLY_SUPPORTED`, with a null
baseline time and no percentage speedup. Raw logs and failure diagnostics remain
in the A/B artifacts. A candidate failure aborts rather than triggering retries.

Session initialization required by a benchmark belongs in its `setup`.
Standalone test cases without `threshold_ms` are not executed in A/B mode.
Every A/B fixture retains its complete MemCP server log before runner-owned
cleanup, including fixtures whose individual queries succeeded. The final
cross-fixture verdict is computed later, so query-success logs are needed to
investigate maintenance activity around a failed timing comparison.

## Document application SQL coverage

Keep document-management application regressions in the existing suites below;
reuse their synthetic fixtures rather than copying production dumps, customer
identifiers, filenames, sessions or credentials. These suites are discovered by
normal CI. No separate application server or session cookie is required for
this SQL correctness coverage.

| Application path | Existing suite |
| --- | --- |
| Search counts, hierarchy menu/folders, nested tenant/site permissions | `planner/subqueries/navigation-permission-probes.yaml` |
| Per-user folder options from original `COALESCE(SUM(CASE ...) > 0, TRUE)` and paired `NOT EXISTS OR EXISTS`, distinct flags and changed memberships | `planner/subqueries/navigation-permission-probes.yaml` |
| Direct file access through DAV/document `EXISTS (UNION ALL)`, guest denial, grant/revoke and independent sessions | `planner/subqueries/navigation-permission-probes.yaml` |
| Upload metadata, delayed text extraction, file rename, document removal and repeated search | `planner/subqueries/navigation-permission-probes.yaml` |
| Selected/inverted document sets, composite selection identity, NULL items and 4,096-item selections | `planner/subqueries/navigation-permission-probes.yaml` |
| Ordered current page, obsolete/unseen notifications and insert-select acknowledgement | `integration/query-shapes/notification-membership-workflow.yaml` |
| Notification ordering/anti-joins across shards and lookup mutations | `performance/notification-membership-scaling.yaml` |
| Session-dependent grouped reminder counts | `planner/aggregates/session-group-reminders.yaml` |
| Tenant membership, LIKE search/count and ordered ACL pages at scale | `performance/tenant-document-membership.yaml`, `performance/like-acl-query-shapes.yaml`, `performance/ordered-acl-page-window.yaml` |
| 800k-document navigation, empty years and complete composite membership keys | `performance/tenant-document-navigation.yaml` |
| 800k-reference point permissions without unique metadata, duplicate driver rows and scalar cardinality | `performance/correlated-union-file-acl.yaml` |
| LIKE/index visibility after inserts and deleted selection/carrier rows | `storage/indexes/fulltext-like-index.yaml`, `integration/regressions/deleted-row-carrier-visibility.yaml` |

The stateful tests intentionally execute identical SQL again after mutations or
session changes, checking both newly granted access and revoked access. Run the
performance suites with `PERF_TEST=1` to include their timed cases. HTTP upload,
PDF rendering, ZIP contents, DAV protocol handling and mail delivery remain
application-level checks; these SQL tests do not claim to exercise those layers.

## Reusing prepared A/B data

Query suites may opt in with `metadata.reuse_fixture: true` and
`metadata.restart_after_setup: true`. Their top-level setup must create persistent
fixtures (`ENGINE=sloppy` is suitable for generated benchmark data). The old/base
binary runs this setup once and exits through SQL `SHUTDOWN`; only a successful
process exit publishes the prepared snapshot. Preparation runs no benchmark
queries. The new binary never writes the reusable source snapshot.

Every round runs old first, new second. Each process receives its own ordinary
file copy of the same base-written snapshot, including verification rounds;
there are no hard links, shared writable data directories or reused processes.
This restores the original pre-query catalog, excluding group/cache tables and
indexes produced by earlier benchmark queries, and resets process-local compiled
plans. OS page caches are not explicitly flushed. The opt-in therefore measures
cold query/operator state, not cold OS pages or import speed. Existing case setup,
result assertions, warmup/sample configuration and cleanup still execute.

Cold query indexes remain part of the measurements. If two cases rebuild the
same source merely to obtain independent cold indexes, prepare separate source
tables once and give each case its own untouched table. Do not prebuild indexes
or group state during preparation. All seven verification pairs still rerun
every case, including initially successful cases.

`metadata.performance_shard_weight` is a positive integer used only to distribute
whole suites across seven runners for each native/JIT configuration. The default
is 1. Weights balance measured fixture costs across all seven runners under the
unchanged 15-minute limit. CI artifacts include process startup and both roles;
incomplete verification runs are projected to all seven pairs when updating
their weights. This avoids reserving a runner for a short suite while placing
several longer verification runs together. Weights are relative reservations for
complete cohorts; they only guide scheduling, and source/measurement times remain reported. This changes
neither test coverage nor any query, setup or regression budget. The KPI workload
keeps its measured cold board and the next seven measured boards, without twenty
additional discarded warmups. Normal tenant hierarchy checks use one complete
5000-row period of the sparse keys; performance runs retain the full 800k source
tables, and the normal query, row assertions and time limits remain unchanged.

Calibrate relative shard weights from complete A/B process costs, including
preparation and snapshot clones. Reserve all fourteen processes for every known
verification family, including `unique-insert`; use the larger ordinary/JIT
estimate because both configurations share the same weights. Before balancing,
check that their summed costs fit the runner count times the unchanged deadline:
weights cannot supply missing execution capacity.

The current calibration covers observed verification of
`adaptive-filter-selectivity`, `domain-snapshot`, `group-range-cache`,
`range-first-aggregate`, `temporal-order-output`, `tenant-document-navigation`,
`unselective-scalar-order-limit` and `unique-insert`. Maximum clean observed
old/new process costs, projected to fourteen processes for these eight families
and two for the other suites, total 5659.10 seconds across all 54 suites.
Six 900-second slots cannot fit that envelope, even without orchestration.
The original seven-slot weights project a maximum of 836.87 seconds. The
concurrent single-row UNIQUE case now performs 1024 inserts per writer in one
invocation, preserving four writers, SAFE durability and exact worker counts.
It keeps the 10000-ms threshold, one timing sample and zero warmups. Repeating
unchanged mutation samples would reuse primary keys rather than repeat an
equivalent workload.

Fixed fresh-process observations on one local persistent filesystem measured
128 inserts per writer at 566.94/568.00 ms and 1024 at 4500.78 ms. Charging the
full observed 3933.83-ms increase to each of fourteen verification processes
adds 55.07 seconds to the UNIQUE shard: its projection rises from 800.53 to
855.61 seconds. The resulting envelope totals 5714.17 seconds and leaves
44.39 seconds before the unchanged deadline; the largest observed successful
orchestration overhead was 12.53 seconds. This extrapolates local workload
cost onto historical CI process costs; it does not prove a CI regression's
cause or statistically establish future variance. These are estimates, not a
guarantee for future cost growth or additional suites entering verification.
Preserve the full suite list and measurement protocol when rebalancing.

Suites with identical top-level preparation inputs share one import. Snapshots
are temporary and deleted on success or failure; they are not committed,
uploaded, or placed in GitHub caches. Import, mutation and storage-construction
benchmarks must retain their own setup: a base-written snapshot intentionally
compares readers of the same stored representation, not each revision's writer.
Do not opt in suites that depend on process-local setup bindings or that must
measure a memory engine. Sloppy fixtures survive the explicit graceful shutdown;
they do not acquire production durability guarantees.

The manifest records preparation, copy and process durations, snapshot sizes,
binary/runtime-library hashes and shard assignment. Per-process timing artifacts
separate startup, execution and shutdown/cleanup; server logs and every raw query
sample remain available. If both executable and dynamically loaded runtime
libraries match, report the relative timings as A/A variance rather than an
engine-patch regression. Both processes must still return correct results under the existing request/setup
deadlines. The reference `threshold_ms` does not become a new, unscaled absolute
gate in A/B record mode. Query errors retain the complete verification
protocol; changes to Scheme code prevent the identical-execution shortcut.

CI validates the full discovered suite set before partitioning it across six
separate runners; each suite is assigned exactly once. The original required
check passes only when all shards succeed. Timed queries do not compete on one
runner. Measurement has a 15-minute cap per shard; jobs allow 30 minutes including
cold external PHP/JIT compiler builds. Those toolchains are cached by runtime
ABI/source identity, while benchmark data is never cached. A timeout is a failure,
not an omitted test or a successful partial result. End-to-end 15-minute CI is a
measurement goal, not yet a demonstrated guarantee on a cold cache.

### Shared scan diagnostics and concurrent requests

Functional suites that inspect `system_statistic.scans` declare
`metadata.scan_debugging: true`. The runner enables ScanDebugging once for that
cluster, runs eligible suites concurrently against the same MemCP process and
data directory, waits for every worker, and restores the previous setting even
when a suite fails. The CI hook uses the same cluster declaration. Suites must
filter/delete diagnostic rows by their own fixture tables; they must not toggle
ScanDebugging themselves. Declare `isolated: true` as well when a suite changes
another global setting such as ShardSize or JoinReorderDPBudget. Isolation still
uses the shared server; it is a scheduling barrier.
Suites that reset the complete scan log or assert across generic query-group
names also run exclusively within the cluster. The runner rejects parallel
cluster declarations that truncate the shared scan log. Server-restart suites
stay outside the cluster.

The processlist suite observes two simultaneous HTTP requests at a table-lock
barrier, verifies a third request can progress, then checks each request's own
session-variable bindings and result after release. This runs in both normal
and JIT CI, including alongside other suites in the scan-diagnostics cluster.
The runner contract tests additionally require concurrent cluster workers and
verify setting restoration on success and exceptions. Performance suites keep
their existing fixture, restart and measurement scheduling.

## UUID and cluster INSERT regressions

`sql/compatibility/uuid-{mysql,postgresql,tsql}.yaml` covers UUID generation,
canonical assignments, conversions and per-row defaults in each frontend.
`sql/ddl/auto-increment.yaml` retains the traditional integer allocation contract.
The three-peer shared-storage INSERT/restart oracle is selected with
`python3 tools/replication_cluster_stress.py --cluster-inserts-only --seed 20261027`.
The cluster workflow runs it with ordinary and race-instrumented processes.

## Cluster transport and backend simulation

New suites (`metadata.cluster: true`) and the process-runner CLI default to
fullspeed: zero added latency and unlimited bandwidth. The permanent operator
suite explicitly selects the following conservative synthetic profile:

```yaml
metadata:
  cluster:
    cluster_bandwidth_mbps: 1000
    cluster_latency_ms: 0.1
    storage_backend: FakeRemote
    storage_bandwidth_mbps: 1000
    storage_latency_ms: 5
```

Bandwidth uses decimal Mbit/s (1000 means 125 MB/s). Cluster latency is one-way;
a request/reply crosses two receiving endpoints. These values are reproducible
scenario assumptions, not claims about every Ethernet, Ceph or S3 installation.
For an S3-like high-latency scenario, use e.g. 50 ms backend request latency;
[Amazon's guidance](https://docs.aws.amazon.com/AmazonS3/latest/userguide/optimizing-performance-design-patterns.html)
describes small-object latencies in the tens of milliseconds and emphasizes
concurrency and request sizes.

CLI values override YAML, including explicit zero. Omitted values retain the
suite's declaration; suites without a declaration remain fullspeed. For example:

```sh
python3 tools/replication_cluster_stress.py --cluster-suites \
  --cluster-bandwidth-mbps 1000 --cluster-latency-ms 0.1 \
  --storage-backend FakeRemote --storage-latency-ms 50 --storage-bandwidth-mbps 1000

# Disable artificial network/backend delays on the same suite:
python3 tools/replication_cluster_stress.py --cluster-suites \
  --cluster-bandwidth-mbps 0 --cluster-latency-ms 0 \
  --storage-backend filesystem --storage-latency-ms 0 --storage-bandwidth-mbps 0
```

The server also accepts `--cluster-bandwidth-mbps` and `--cluster-latency-ms`
directly, with zero defaults. Only the receiving side of the native `/cluster`
WebSocket protocol is instrumented, including authentication frames. SQL/SCM
HTTP, local scans and sending are not delayed. Every application frame is
charged for its actual encoded MOSI envelope and payload bytes. Before reading
the next message header, the receiver sleeps only for the transmission time
not already covered by receiving, decoding and dispatch work. All connections share the receiving node's bandwidth budget, so adding peers
does not multiply its simulated NIC rate. This is not a packet-loss simulator. WebSocket/TCP/TLS overhead and keepalive control frames are not counted.

Propagation latency is waited in dispatch workers, so independent messages can
remain in flight concurrently. Resource/schema ordering remains intact. Pending
messages are bounded by the existing 32-MiB receive-work budget; disconnect
cancels and joins outstanding delay timers. Existing application deadlines still
apply under extremely slow profiles. Go timer granularity can exceed a requested
sub-millisecond delay; diagnostics record actual wait, not the requested duration.

The native `filesystem` backend accepts an optional administrator-configured
`root` directory. Its default is the server's `-data` directory. Absolute roots
select that explicit namespace; relative roots stay below `-data`. Each database
still occupies its own validated name below the root. Configuration discovery,
clone and DROP use that same namespace; database-directory symlinks and global
settings collisions remain rejected. Unknown options, non-string/empty roots
and escaping relative roots fail before a configuration is published.

```sql
CREATE DATABASE `memcp-tests` SET backend='filesystem', root='/srv/shared-data';
```

The equivalent backend JSON is `{"backend":"filesystem","root":"/srv/shared-data"}`.
Relative roots reject symbolic links in every existing path component, including
when later components do not exist. Absolute roots remain an explicit administrator
choice of namespace.

The filesystem-root YAML suite declares `filesystem_fixture: root_ancestors`
with `isolated: true`. This bounded runner preset owns one temporary outside
directory and fixed symlink/description names inside `MEMCP_TEST_DATA_DIR`;
YAML cannot supply paths or shell commands. It refuses preexisting fixture names,
retains the fixture through suite restarts and cleans up only its own artifacts.
Cluster participants keep independent private `-data` directories and configure
the same real filesystem root for their shared database; they do not use
per-database symlinks or change the selected storage backend.

`FakeRemote` is a filesystem-backed persistence backend using the ordinary
backend declaration, independently of table ENGINE durability:

```sql
CREATE DATABASE `memcp-tests` SET backend='FakeRemote',
  latency_ms='50', bandwidth_mbps='1000';
```

Its `prefix` selects the physical database directory (relative to `-data`, or an
absolute path); the default is the database name. Cluster fixtures point every
participant at the same existing shared directory. JSON configuration accepts
numbers or SQL-style numeric strings. `latency_ms` and `bandwidth_mbps` default
to zero. Each backend instance shares one aggregate transfer budget across its
read/write streams. Each object/list/open/flush operation pays request latency;
64-KiB buffering avoids adding a timer for every serialized scalar. WAL writes
retain native encoding and are charged at flush; replay pays for the existing
WAL object before native bounded decoding. Aborts discard private buffered bytes.
Filesystem formats, atomic schema publication, fsync and SAFE/LOGGED/SLOPPY/MEMORY
semantics are retained. The backend models delay/throughput, not S3 consistency,
Ceph quorum behavior, retries or failure distributions.

Every YAML testcase report records `operation_wall_ns` and per-node `io` counter
deltas sampled around the body, outside setup, EXPLAIN and cleanup. `(io_stats)`
returns process-local `cluster_receive` and `fake_remote` counters: `operations`,
`bytes`, and actual injected `wait_ns`. Fullspeed transport still records bytes
and messages, without artificial sleeps. Reports and CLI reproduction commands
retain the selected profile. Concurrent background activity can contribute to
these diagnostic counters; they are not transaction accounting.

Summed wait across connections/nodes can exceed elapsed wall time because waits
and CPU work overlap. Do not call `sum(wait_ns)/operation_wall_ns` a synchronization
overhead percentage. These diagnostics prepare controlled local/remote and
fullspeed/throttled comparisons; they do not introduce a new performance gate
or change shard sizing. Larger shards amortize fixed per-job costs, while
batching amortizes per-message costs; per-byte transfer/serialization costs do
not disappear with larger shards. Later comparisons should keep fixture,
shard count, concurrency and repetitions fixed and retain cold and warm phases.


`sql/compatibility/tsql-virtual-commands.yaml` covers the restricted command
adapter for legacy `xp_cmdshell` calls. It is not a process runner. Fully
anchored, case-insensitive Scheme patterns recognize only `DIR <path>`,
`COPY <source> <target>` and `DEL <path>` with bounded quoted or unquoted
arguments. Expansion, chaining, redirection and control bytes are rejected.
Each path goes through the shared virtual path parser. The dispatcher takes
concrete engine operations as an explicit argument; no command registry or
host fallback exists. Currently only DIR has an implementation and lists the
actual declared virtual children with NVARCHAR output. It reports no disk
capacity, volume identity, physical files or German free-byte summary. The
compatibility call requires SQL administrative permission. COPY and DEL fail
because no native snapshot/artifact capability is available; they do not
create or remove files and never return hollow success. SQL BACKUP/RESTORE
remain unsupported until a coherent native backup/clone contract is supplied.

`storage/persistence/replicated-catalog-authority.yaml` runs the same assertions
standalone and through the two-process replication regression harness. Its
`replication_client: master|slave` hint selects the owning endpoint only in that
harness; the ordinary runner executes every case against its single server.
The harness uses its ordinary `replica_test` domain so frontend startup's
`memcp-tests` planner fixtures never access a replica before its manager starts.
SQL selects that domain through the runner's existing database argument; native
SCM uses its existing `{database}` fixture placeholder. The ordinary runner uses
`memcp-tests`. Query shapes and assertions remain identical.
The harness additionally checks the persisted database incarnation and native
catalog decision watermark after authenticated schema return and restart.

### T-SQL unary plus vendor compatibility

The neutral [SQL Server 2022 CU26 reference](https://dbfiddle.uk/DLSkEqAN)
(version 16.0.4265.3 Express on Linux) preserves DATETIME, SMALLDATETIME,
DATETIME2(6), NVARCHAR(8), and typed NULL operands under unary plus. The same
reference creates and invokes a scalar function assigning `+DATEADD(...)` to a
local DATETIME and returns the expected difference. This observed identity
behavior is broader than the numeric-only argument description in the
[Microsoft unary-plus reference](https://learn.microsoft.com/en-us/sql/t-sql/language-elements/unary-operators-positive).
The frontend retains the operand formula, declared type/dimensions and collation;
it does not coerce temporal values to numeric carriers or reinterpret text.
Other nonnumeric families and unary minus are not established by this reference
and are not expanded by this compatibility change.

### Relational scalar source planning

Read-only straight-line scalar modules with DECLARE, SET, SELECT assignments and
RETURN use a separate frontend language whose native callable is a planning
factory. An authorized immutable source/declaration snapshot expands to ordinary
query blocks before shared binding and decorrelation. No SQL executes in a row
callback and no request session or transaction is retained in the factory.

The compiler exports one bounded parameter/local/context tuple per logical
state, behind an ordinary global aggregate boundary. SELECT assignment projects
one presence-and-values packet per actual input row and reduces whole packets
with an unordered last-value aggregate; nil is the identity for empty partials.
An empty stream retains the previous locals; a present packet containing NULL
overwrites them. An actual TOP 1 retains its original window directly. A scalar
aggregate such as COUNT(*) supplies a present row even on empty input. Native
source scans, joins, own-transaction visibility and row limits stay in the
shared planner, rather than a procedural per-row query executor.

The initial invocation profile admits already compatible immutable column,
parameter and literal arguments. Compound or volatile argument domains,
unbounded ordered assignment, larger TOP windows, relational control flow and
nested source-function calls need further logical operators/proofs; this does
not claim their language syntax is invalid. The declared SQL signature remains
unchanged and DATEFORMAT/LANGUAGE are explicit scalar callsite operands.
