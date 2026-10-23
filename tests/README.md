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
driver. Each probe keeps one connection for its statements, supports named
parameters and prepared calls, and checks rows, values, column names and errors.
Unsupported frontend features remain noncritical specifications.

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

The **Replication Cluster Stress test** workflow executes this runner directly.
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

The existing `affected_rows` path remains unchanged for compatibility with the
current SQL endpoints. It compares a supplied completion count, but an absent
count is not yet rejected. Therefore an `affected_rows` expectation alone does
not prove the result of a command returning an empty response. Keep state and
schema readback assertions; do not describe these legacy checks as complete
completion-protocol coverage. Strict completion validation and its additional
runner contracts must land together with the corresponding frontend/API output
contract. This limitation does not justify inventing a count, replaying a write,
or demoting a SQL specification.

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
