<!-- Copyright (C) 2026 MemCP Contributors; SPDX-License-Identifier: GPL-3.0-or-later -->

# Adaptive decoded blob RAM

External values remain gzip-compressed in the existing content-addressed format.
The 2 KiB inline boundary, planner, query-cache guards and costgen weights are
unchanged. `OverlayBlob` optionally retains successfully decoded strings.

## Existing hierarchy and admission

Index Savings already expresses reuse, and CacheManager provides budgets,
telemetry and partial/full eviction. Ordinary cachemap entries admit on first
access and synchronize/register each entry. Materialized string dictionaries
also decode completely on first access. Neither implements blob admission. The
existing `OverlayBlob.values` map contains serialized gzip/build data, so decoded
RAM values cannot reuse it without mixing representation and persistence.

One column-generation-local `cacheObject` therefore owns decoded values and
probation metadata. It registers as one ordinary `CacheEntry` in the existing
manager. The first successful read records the hash and measured reload time,
without retaining the payload. A second read can retain it. Repeated identical
hashes within a batch are actual reuse and deduplicate within that owner; a
scan of distinct new hashes retains only metadata.

Telemetry is bounded reuses × measured reload nanoseconds / charged resident
bytes. Reload measurements include file access, gzip and allocation. Failed
reloads never train the estimate. At most eight reuses are credited. Partial
eviction drops payloads at or below the owner's mean benefit/byte, halves the
retained histories and requires fresh evidence for readmission. Full eviction
also releases the map and probation metadata.

There is no idle expiry or time-based cooling of blob admission history. Warm
entries survive an idle weekend without pressure. Under pressure the existing
manager still combines idle age and telemetry. This remains a heuristic across
cache types, not a globally optimal common latency model.

The manager now collects at least 16 available candidates before comparing
scores. Previously one large warm owner could alone fill the byte target,
preventing comparison with much cheaper small probation owners. Probation-only
owners offer complete release in the partial pass because they have no decoded
representation to preserve. Type weights and expiry settings are unchanged.

## Concurrency, lifetime and memory

Synchronization and usage publication happen once per batch of at most 256
rows; a scalar point read is a one-value batch. Inline-only reads avoid cache
locking. Generated JIT bulk getters use the same resolver. Cold decoding is
serialized per column owner, without a database-wide lock.

Size publication is queued while locked, but the manager processes it only
after unlock. Otherwise a newly added, busy probation owner could be skipped
and warm contents stolen instead. Callbacks use `TryLock`, acquire no shard or
table locks and call no public CacheManager methods. Panic cleanup publishes
previous successful admissions and releases the lock.

Owners retain no source-column, shard, database or persistence pointer. Updates
use the existing delta/generation visibility rules; exclusive source replacement
resets the owner. Different owners charge their own copies rather than assuming
shared lifetime from a hash. Retired owners remain charged until pressure removes
them, but cannot keep a persistent generation alive. Returned immutable Go
strings remain valid for active readers after eviction.

Payloads are charged separately from base-column storage. Each hash reserves
256 bytes for entry/map capacity and growth; each registration reserves 512
bytes for manager/map overhead. Partial release never subtracts buckets Go
still retains. Cloned strings avoid retaining excess Builder capacity. Small
strings reserve a conservative 25% allocator allowance rounded to 64 bytes;
strings above 32 KiB round to 8 KiB heap pages. The fixed cache header is in the
parent-column estimate.

A miss samples the configured RAM budget once per batch. Admission cannot grow
one owner's payload beyond that entire configured budget. Denied reusable
values request room through the existing manager after unlock. Probation and
concurrent batches still follow its soft-budget semantics. Gzip/Builder buffers,
output buffers and active-reader references are query working memory. Physical
reclamation follows reader release and Go collection; cache ownership is not an
RSS limit. Memory measurements below distinguish these quantities.

No cache operation deletes a persistent blob or changes its reference count.
Complete reference proof remains the deletion authority. Misses still report
missing or corrupt payloads and can succeed after repair. Hits return previously
fully decoded content; they do not reopen immutable files to detect external
filesystem changes. Legacy ambiguous references retain their original verified
path. Recovery never depends on the RAM cache.

Partial streaming decode and early comparison/matcher exit require a separate
checksum/error contract and are outside this implementation.

## Validation and A/B method

Baseline: master `6a8642977`, including blob safety fix #838. All fixtures are
synthetic; no live service or production data was used. Full tests run in CI.

Permanent cases extend `tests/storage/formats/blob-storage.yaml`: actual external
overlays, repeated text/binary reads, shared hashes in rows/tables, update and
failed-write visibility, surviving ownership, restart, parallel readers and RAM
pressure. Performance cases project LIKE/length per row, avoiding aggregate
keytables which would bypass content reads on warm runs.

Temporary Go probes additionally checked admission, a one-shot scan against a
warm owner, source replacement, missing/corrupt gzip followed by repair, and
concurrent admission/eviction with the race detector. The one-shot reproducer
originally displaced about three quarters of a warm owner's payload charge;
the corrected comparison/publication protocol preserves it.

Microbenchmarks use Go 1.24, linux/amd64, GOMAXPROCS=4 and 128 distinct strings
of 512 B, 4 KiB or 64 KiB. Strings have `row-%08d:` prefixes and either repeated
ASCII or deterministic pseudorandom printable ASCII (seed 42). External values
use the normal default gzip encoder and filesystem blobs. Both revisions use
the same files/seed, three warmups and 32 measured iterations per case, run in
A/B/B/A order. Tables report the mean of each revision's two blocks. Separate
inline checks use two blocks of 10,000 iterations. “Cold” means a fresh decoded
owner before each iteration, with filesystem pages already warm.

Workloads are scalar point reads, range scans, scans with a changing literal
search term, alternating two owners, and cold scans. Budgets are 128 KiB and
64 MiB. Allocation metrics include the benchmark's small search-pattern string.
Memory probes perform twelve scans, sample heap/RSS every millisecond and drop
caller-owned output before measuring retained heap. Sampled peaks are observed
lower bounds, not exact maxima. CPU includes the sampler's overhead.

JIT SQL measurements use the existing jit-foreign-frames Go toolchain, the same
256-row fixture as the YAML suite, three warmups and seven measured queries per
case, again in A/B/B/A order. The fixture contains numeric prefixes followed by
8 KiB of repeated text. HTTP and result encoding are included.

## Results (2026-09-09)

AMD Ryzen 9 7900X3D. Values below are baseline → candidate. The matrix had no
external-blob case slower by 20% or more; tiny inline point reads added about
1.5 ns in the longer verification. HTTP point-read changes remain noise-sized.

### JIT SQL, milliseconds per query

| Projection | Master | Candidate | Change |
|---|---:|---:|---:|
| Repeated LIKE projection over external blobs | 5.448 | 1.155 | -78.8% |
| Repeated full content length projection | 5.605 | 1.011 | -82.0% |
| A different LIKE pattern reuses decoded content | 8.763 | 4.397 | -49.8% |
| Repeated point read of external content | 0.796 | 0.818 | +2.7% |

Each SQL block uses the median of seven queries; the table averages the two
blocks for each revision. The fixed fixture always has 256 rows.

### Blob reader matrix

Point times are **µs**, all scan times are **ms**. “Change” scans include literal
matching with a different row-prefix search string each iteration; “alternate”
switches two independent column owners. “Cold” does not include disk-page eviction.

| Text | Budget | Point µs | Range ms | Change ms | Alternate ms | Cold ms |
|---|---|---:|---:|---:|---:|---:|
| 4 KiB repeated | 128 KiB | 25.665 → 0.093 | 2.258 → 1.746 | 2.309 → 1.682 | 2.155 → 1.919 | 1.977 → 1.854 |
| 4 KiB repeated | 64 MiB | 14.774 → 0.099 | 2.091 → 0.003 | 2.309 → 0.008 | 1.934 → 0.003 | 2.055 → 1.838 |
| 4 KiB random | 128 KiB | 41.566 → 0.075 | 4.664 → 4.302 | 4.771 → 4.493 | 4.739 → 4.627 | 4.840 → 4.684 |
| 4 KiB random | 64 MiB | 36.157 → 0.095 | 5.073 → 0.002 | 5.157 → 0.039 | 4.928 → 0.003 | 4.696 → 4.654 |
| 64 KiB repeated | 128 KiB | 51.621 → 0.099 | 5.820 → 5.639 | 6.378 → 5.287 | 5.334 → 5.206 | 5.425 → 6.122 |
| 64 KiB repeated | 64 MiB | 46.239 → 0.079 | 5.144 → 0.003 | 5.673 → 0.090 | 5.228 → 0.003 | 5.107 → 6.077 |
| 64 KiB random | 128 KiB | 374.779 → 0.073 | 48.584 → 49.455 | 50.305 → 50.459 | 48.922 → 48.853 | 49.589 → 49.143 |
| 64 KiB random | 64 MiB | 390.627 → 0.075 | 48.875 → 0.002 | 50.219 → 1.243 | 48.986 → 0.003 | 49.355 → 49.264 |

The weakest cold result above is 64 KiB repeated text with a 64 MiB budget:
5.107 → 6.077 ms (+19.0%). A longer, identical 128-iteration A/B/B/A follow-up
measured 7.014 → 6.963 ms (-0.7%). Cold timings are noisy on this shared host;
there is no claim of a cold-read speedup, and first-read metadata has a cost.

For the 512 B inline verification, median point/range/change/alternate times
were respectively 9.36/813.85/4275.5/843.45 ns on master and
10.90/677.8/3914.0/659.85 ns on the candidate. Inline cache charge stayed zero.

### Range-scan allocations and cache charge

Allocation bytes are MiB per operation, followed by allocation count. The cache
charge is a conservative owner estimate after the measured block, not RSS.

| Text | Budget | Allocation MiB / count, A → B | Candidate cache KiB |
|---|---|---:|---:|
| 4 KiB repeated | 128 KiB | 5.128 / 1694 → 4.888 / 1650 | 75.0 |
| 4 KiB repeated | 64 MiB | 5.128 / 1694 → 0.000 / 0 | 672.5 |
| 4 KiB random | 128 KiB | 5.101 / 1712 → 4.841 / 1658 | 75.0 |
| 4 KiB random | 64 MiB | 5.116 / 1715 → 0.000 / 0 | 672.5 |
| 64 KiB repeated | 128 KiB | 18.615 / 1828 → 18.695 / 1868 | 32.5 |
| 64 KiB repeated | 64 MiB | 18.625 / 1828 → 0.000 / 0 | 8224.5 |
| 64 KiB random | 128 KiB | 18.589 / 1944 → 18.654 / 1982 | 32.5 |
| 64 KiB random | 64 MiB | 18.588 / 1942 → 0.000 / 0 | 8224.5 |

### CPU and resident/peak memory for twelve scans

Heap values are MiB above the pre-read baseline; retained heap is sampled after
clearing caller output and running GC. Peak heap/RSS are sampled during reads.
RSS columns show the whole process after output release and the sampled peak.

| Text | Budget | CPU ms A → B | Retained heap MiB A → B | Peak heap Δ MiB A → B | RSS after MiB A → B | Peak RSS MiB A → B |
|---|---|---:|---:|---:|---:|---:|
| 4 KiB repeated | 128 KiB | 49.99 → 49.74 | 0.13 → 0.14 | 3.06 → 3.14 | 23.47 → 22.98 | 23.65 → 23.47 |
| 4 KiB repeated | 64 MiB | 50.07 → 10.15 | 0.09 → 0.60 | 2.97 → 2.26 | 24.10 → 22.77 | 24.10 → 22.73 |
| 4 KiB random | 128 KiB | 109.66 → 83.62 | 0.04 → 0.10 | 2.64 → 3.00 | 22.51 → 22.64 | 22.76 → 23.26 |
| 4 KiB random | 64 MiB | 92.14 → 18.05 | 0.04 → 0.60 | 2.58 → 2.21 | 22.29 → 22.85 | 22.75 → 22.85 |
| 64 KiB repeated | 128 KiB | 141.67 → 125.66 | 0.04 → 0.06 | 19.87 → 20.92 | 33.48 → 33.20 | 40.75 → 40.94 |
| 64 KiB repeated | 64 MiB | 111.43 → 32.26 | 0.09 → 8.11 | 19.15 → 17.31 | 36.16 → 32.88 | 39.75 → 37.88 |
| 64 KiB random | 128 KiB | 652.71 → 683.44 | 0.04 → 0.06 | 20.76 → 20.61 | 34.16 → 33.26 | 41.60 → 41.06 |
| 64 KiB random | 64 MiB | 662.32 → 120.27 | 0.04 → 8.06 | 20.20 → 14.62 | 33.18 → 32.29 | 40.26 → 34.89 |

Pre-read RSS was 19.3–19.8 MiB across these processes. Warm cache residency trades
additional retained heap for much less decode/GC churn: the 64 KiB random, 64 MiB
budget probe retained about 8 MiB more heap, while twelve-scan CPU fell from
662 to 120 ms and sampled peak RSS from 40.3 to 34.9 MiB. At 128 KiB, those large
scans remain effectively uncached; no whole 8 MiB decoded scan is admitted.

These measurements do not imply the same speedup for a complete application:
LIKE/collation work, network output, already cached aggregates, and planner choices
can dominate. Simultaneous owners still compete through a bounded candidate
window, and a first scan has nonzero metadata cost.

To reproduce the permanent JIT integration cases with an existing `./memcp` build:

```sh
PERF_TEST=1 python3 run_sql_tests.py tests/storage/formats/blob-storage.yaml 18494 --fail-fast
```


Local final checks: 83/83 blob YAML cases with JIT, 5/5 undercount-recovery
cases, 32/32 memory-pressure cases, targeted existing Go blob/cache tests, and
the temporary race/admission/fault-repair probes passed. All four JIT A/B
projection cases passed on both revisions in both orderings.

`EXPLAIN`, `EXPLAIN IR`, `EXPLAIN PHYSICAL` and `EXPLAIN REORDER` were checked
for the length projection: the logical form is one query-block and the lowered
code is one scan reading `id` and `payload`, with `strlen` in the output mapping.
There is no aggregate keytable bypassing payload access in that measurement.
