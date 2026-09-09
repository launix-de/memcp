<!-- Copyright (C) 2026 Carl-Philip Hänsch; SPDX-License-Identifier: GPL-3.0-or-later -->
# Adaptive selectivity and the Omnestum search

## Representation and remaining gaps

The planner consumes a scalar probability for filtered cardinality and cost.
The underlying model need not be scalar. Currently it retains exact expression /
bound-value keys and LIKE families separated by column, collation and wildcard
topology. Eligible patterns also carry literal rune lengths. An unknown pattern
uses averages of retained patterns at the same length, linear interpolation, or
extrapolation over at most four characters with the existing 0.35 decay slope.
Exact reads are O(1); histogram reads visit at most 64 immutable entries.

This is a bounded workload-derived histogram, not a data-distribution histogram.
The shared 64-slot budget can evict samples; same-length words can have unrelated
frequencies. Histogram confidence 0.35 is heuristic, not a calibrated error bound.
Repeated executions of one retained word do not add histogram weight. Exact
expression/value estimates are preferable for repeated searches. Numeric/date
ranges would need a rebuild-collected value CDF; categorical equality needs
frequencies/heavy hitters and distinct counts, not numeric interpolation.

## Evidence from the actual query

The original roughly 29 KB SQL was extracted from PM2 TracePrint, retaining its
session binding: filename LIKE and fulltext search for `carglass`, file-to-document
membership, tenant/access checks, sorting, LIMIT 72. The unchanged query returned
72 rows in 690.395 and 669.333 ms with identical result hashes. The running modified
binary predated #837, so these timings are not an A/B test of adaptive feedback.

Separate indexed operator measurements on that instance:

| Operation | Distinct output | Cumulative wall time |
| --- | ---: | ---: |
| Filename LIKE | 105 files | 2.83 ms |
| Fulltext predicate | 92,803 files | 539.12 ms |
| Union | 92,822 files | 545.23 ms |
| Projection to documents | 97,424 documents | 571.19 ms |

These are separate cumulative measurements, not additive timings. With about
785,893 files, the same eight-character word has approximately 0.0134% filename
selectivity and 11.81% fulltext selectivity. These must not share a length bucket.
There are 86 overlapping files: adding branch counts would overcount them.
Projection fanout is about 1.050 documents per matching file, not a probability.
The probability of passing search AND a user's ACL within an ordered prefix
needs a separate population and the relevant tenant/user bindings.

EXPLAIN PHYSICAL showed an uncertain projection interval [0, 855109] crossing a
637-row plan boundary, with a 100 ms observation budget. Runtime prepared the full
projected candidate set before dispatch. Fixing that batch-accept/preparation
policy is a separate task. This change provides local filter evidence, not a
claimed fix or measured speedup for the complete application query.

## Collection and persistence

Previously ordinary full scans could train feedback, but complete RecSet builds
could not. Now table-input `scan_recset` reuses the builder's distinct output count
and visible shard population. Restricted RecSets, pseudo-column restrictions,
ACID snapshots, unique points and internal tables are excluded. No new element
counters, histogram updates or filter-loop synchronization are added. One
best-effort CAS publishes after shard completion; one bounded table merge follows.
An identical population/rate retains the existing immutable shard and table
snapshots instead of allocating and publishing redundant updates. Counting deletions visits bitmap words once per completed shard, so that part is
not strictly O(1). Feedback adds no read lock.

Shard-local EMA state remains volatile. A first complete observation in a new
generation replaces its prior; subsequent observations mix 99% old / 1% new.
Table merges weight shards by population, retaining the original prior for
unobserved shards. Compatible older table aggregates remain available as
historical estimates until replaced. They are not copied into all shards as if
those shards had been measured.

Existing schema checkpoints now optionally persist a versioned `filter_feedback`
object with at most 64 table aggregates and their LIKE family/length metadata.
Scans neither write schema nor set its dirty flag. Startup restores historical
estimates at confidence 0.35 without loading shard columns. Column names, types,
dimensions and collations must match. Old schemas need no migration; unknown
versions and invalid optional hints are ignored. Memory/cache tables omit hints.
Uncheckpointed observations can be lost without affecting row durability.

Existing plan-cache statistics guards read feedback classes. New keys, changed
geometric selectivity classes or changed coverage invalidate cached costing.
These are table-wide power-of-two classes, not exact plan-specific crossover
guards. A crossover within one class can still go unnoticed.

Cached EXPLAIN output now uses the same table-statistics dependencies, without
executing candidate preparations while checking its cache. An existing text
fixture verifies the identical EXPLAIN REORDER query changes from the cold 1%
prior to the learned 8/2048 = 0.390625% rate. EXPLAIN COMPILE still bypasses caching
so it measures an actual compilation.

## Verification

Tests cover full/restricted RecSets, unique points, shard EMA and weighted merges,
concurrent publication/checkpoint serialization, invalid optional metadata, schema
compatibility, and a real checkpoint/reload with shards remaining cold. SQL tests
verify indexed RecSet feedback reaches the planner and its cache guard. The
performance fixture uses 2,000 text dimension rows and 20,000 referencing fact
rows; it contains no private application records.

Manual A/B on 2026-09-09 compared baseline `6a8642977` with this change using the
same Go toolchain, fixture, query, warmup and sample counts. The complete RecSet
microbenchmark used 65,536 rows, GOMAXPROCS=1, CPU 22, five alternating runs of
500 iterations after fixture warmup:

| Measurement | Baseline | Change | Difference |
| --- | ---: | ---: | ---: |
| Complete RecSet, median | 3.202718 ms | 3.277633 ms | +0.074915 ms / +2.34% |
| Allocations per call | 69 | 71 | +2 |
| Bytes per call, median | 20,630 | 20,795 | +165 |

The text-membership SQL fixture used normal GC, two isolated data directories,
GOMAXPROCS=1 per process, 10 warmups and 200 timed executions per revision,
alternating request order. Each process had its own CPU (22/23); the second
round swapped CPUs:

| SQL measurement | Baseline | Change | Difference |
| --- | ---: | ---: | ---: |
| Round 1, mean wall time | 20.112562 ms | 20.305865 ms | +0.96% |
| Round 2, mean wall time | 23.886269 ms | 24.154979 ms | +1.12% |
| Combined mean wall time | 21.999416 ms | 22.230422 ms | +1.05% |
| Total process CPU, both rounds incl. warmups | 10.73 s | 10.92 s | +1.77% |

All SQL outputs matched. Inspection of generated code, IR, physical plan and
reorder output found no plan change for this fixture. Earlier trials pinned both
processes to the same CPU; background GC in one process interfered with the
other's request and produced large, inconsistent wall-time differences. Those
trials do not establish a planner regression or speedup. The separated-CPU runs
above retain normal GC; disabling GC was only a diagnostic experiment.

The demonstrated improvements are coverage (full RecSet feedback), restart
availability (the checkpoint test retains a measured 12.3% rate before any shard
loads), retention of LIKE buckets, and no redundant publication for unchanged
observations. No end-to-end speedup of the private application query is claimed.

The EXPLAIN cache follow-up was also measured with the same membership fixture,
normal GC, separate CPUs, 10 warmups and 200 timed cached EXPLAIN REORDER calls:
0.345551 ms baseline versus 0.350319 ms after the change (+0.004769 ms / +1.38%).
Schema persistence goes exclusively through the backend-neutral ReadSchema /
WriteSchema byte interface (FileStorage, S3Storage and CephStorage), with no new
backend-specific operation.

After integrating master `ae5c20c5d` (#840), its absent-search regression exposed
an existing cache-miss bug: a newly compiled variant could consume an observation
whose preparation existed only in that new variant. The miss path now prepares
the chosen variant outside its compile lock before execution, using an explicit
session/transaction procedure boundary. This also avoids evaluating prepared
ASTs in the helper's unrelated lexical frame. A focused test fails before the
fix; the #840 batch/ACL suite passes afterward, including the empty result.

A further manual A/B against `ae5c20c5d`, using the same SQL fixture, normal GC,
10 warmups, 200 samples per revision, separate CPUs and then swapped CPUs, gave:

| Integration measurement | Master | PR | Difference |
| --- | ---: | ---: | ---: |
| Round 1, mean | 22.558117 ms | 21.188738 ms | -6.07% |
| Round 2, mean | 24.012778 ms | 22.988399 ms | -4.27% |
| Combined mean | 23.285448 ms | 22.088569 ms | -1.196879 ms / -5.14% |
| Process CPU, both rounds incl. warmups | 10.84 s | 11.16 s | +2.95% |

Outputs matched. Wall time improved in these two rounds while process CPU rose
slightly; this is an integration overhead check, not evidence of a better plan
for the application query. Focused validation after integration: adaptive
feedback 8/8, batch observation 7/7, prepared statements 57/57, range scans 114/114.

The full SQL CI also exposed diagnostic/calibration paths which had treated a
prepared observation as an opaque session read. Physical diagnostics now resolve
its registered producer as data. Calibration carries only reachable preparations,
in dependency order, in its measured executable plan. Forced prefiltered carriers
retain their own producer rather than substituting the normal prepared carrier.
The original operator/result assertions are unchanged: IN subqueries 73/73 and
indexed membership costing 20/20 pass; a new diagnostic test checks non-execution,
quoting, dependency order, deduplication and omission of unreachable preparations.

One CI cold wide-integer aggregate sample rose from 51.374 to 155.186 ms. A manual
follow-up used the unchanged 60,000-row, 16-column fixture, normal GC,
GOMAXPROCS=4 on CPUs 22–25, six fresh processes/data directories per revision,
alternating revision order, zero warmups and one measured first fill per process:
master `ae5c20c5d` median 47.987811 ms, PR 49.857372 ms (+3.90%); means 53.450502
and 54.048901 ms (+1.12%). Outputs matched, and first-fill variation occurred on
both revisions. This does not reproduce the CI-sized slowdown; the CI threshold
and fixture remain unchanged and must pass on the final revision.
