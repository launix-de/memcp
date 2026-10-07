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

Place a regression at the layer that owns its root cause, not at the layer
where an application happened to expose it. Use `integration/query-shapes`
only when the composition of otherwise independent SQL features is the subject
of the test. Keep one topic per suite; split a file when setup and assertions
cover independent planner or execution contracts.

Run one suite directly with, for example:

```sh
python3 run_sql_tests.py tests/planner/subqueries/deep-correlation-membership.yaml
```

Run a taxonomy section through the pre-commit selector:

```sh
./git-pre-commit 'tests/planner/subqueries/*.yaml'
```

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
The total may be at most 20% slower; no warmup bonus or fixed jitter allowance
extends that limit. Suspect totals receive the usual complete fresh-fixture
ABBA verification, with medians taken across whole workload totals. Independent
latency tests keep their existing median-per-request policy.

- `sql/dml/insert-values-template.yaml`: Bulk INSERT literals, session bindings and computed cells.
- `performance/bulk-insert-compile.yaml`: Cold compilation of parameterized bulk INSERT rows.
- `performance/jit-scan-kernel-compile.yaml`: Full column scans including per-invocation filter and reducer compilation; one cold request plus three repetitions.

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
initial A/B pair plus six additional fresh fixtures per revision (`ABBA` three
times). Every candidate sample must satisfy the result assertions. Every
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
