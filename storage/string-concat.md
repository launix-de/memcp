<!-- Copyright (C) 2026 Carl-Philip Hänsch -->
# Direct string operations on concat pools

`StorageString` version 6 replaces runtime LZ4 dictionaries with immutable pools
of literal byte spans and concat entries. The codec implementation is in
`storage/stringconcat`, shared by storage and Scheme without an import cycle;
`storage/StringConcatCompression.go` owns column integration and legacy loading.

A Scheme value is a 16-byte `Scmer`: a GC pointer to the pool and a 56-bit entry
offset (the remaining eight aux bits identify the tag). An entry starts with its
byte length, followed by `(literal offset, byte length)` pairs and word zero.
Word one selects an inline raw value of the declared length followed by zero.
Words use unsigned base-128 continuation encoding. Literal payload offsets zero
and one are reserved; the first entry lives at offset zero, before its payload.
NULL remains a distinct SQL value, independent of an empty string entry.

The private builder indexes byte trigrams and searches their literal offsets
for the longest reusable span. One- and two-byte pieces remain literals when
reference metadata would cost more. Encoded sequences use a fixed stack buffer
and grow with ordinary append only on overflow. The final immutable pool keeps
neither the build index nor a decoded copy. References address literal spans,
never recursively nested concat entries, so cursors need no reference stack.

Length, handle identity, equal-length rejection, shared literal spans and
unchanged operations have direct shortcuts. Comparisons, LIKE, hashes and UTF-8
operators traverse borrowed spans; a rune may cross span boundaries. A single
span casts to a Go string without a copy. Fragmented string casts allocate their
result; append and streaming output write directly into caller-owned output.
PHP cells, SQL JSONL and BSON scalar output avoid an intermediate Go string.
Numeric coercions inspect their numeric prefix rather than materializing a
whole query/report. Scalar column reads do not update shared metadata; telemetry
updates occur once per batch.

Binary and existing general-ci semantics use cursor comparisons. Full locale
collations retain x/text semantics, including numeric ordering and contractions:
single spans are borrowed; fragmented text appends into reusable collator-owned
output. This temporary collation input is not a column/dictionary cache.
No-op case conversion, trimming, replacement and whole-value substrings retain
handles where possible. A substring within one literal borrows that literal.

## Persistence

The version-6 body contains row count, values and starts storages, then mode:
mode zero stores the inner-format lengths and raw dictionary bytes; mode one
stores only the concat pool, with starts identifying entry offsets. Lengths in
concat headers are not duplicated in a column length directory. Existing
nibble/UUID/Base64 inner encodings remain supported as raw compact data.

Historical versions 0 through 5 and the ASCII raw sentinel remain readable.
Legacy monolithic/block LZ4 is decoded only during private loading and normalized
before publication. Every current save writes version 6. Runtime readers never
register materialized dictionary owners. The old `StoragePrefix` implementation,
which was never produced and had no serializer, has been removed. No persistence
or eviction path gains disk deletion. The upgrade pin and `up_concat_text`
fixture exercise old data, NULL, empty text, shared fragments and restarts.

## Manual three-way measurements

Baseline: `bdaa313a6`. Raw mode differs only by disabling the `finish` call to
`CompressDictionary`; raw disk size uses the explicit V3 raw-fixture writer.
Fixture: 4,096 distinct 482-byte SQL-shaped trace strings, including Unicode;
1,974,272 source bytes. All phases use exactly the same fixture and result checks.
LZ4 was measured first, then raw, then concat. This is a direct-handle Go probe,
not a claim about end-to-end SQL sort speed: the old SQL index builder could
amortize a full LZ4 decode, while the new implementation retains no decoded cache.

AMD Ryzen 9 7900X3D, native Go. Access probes: 20,000 iterations per sample;
build/sort: three iterations per sample. Three samples per mode, no forced GC;
all samples, including the first, are retained below. Sorting starts from a
fresh identical permutation and verifies its complete result each iteration.

| Metric (median) | LZ4 | Raw | Concat |
|---|---:|---:|---:|
| Build | 12.499 ms | 10.238 ms | 14.013 ms |
| Random binary comparison | 4695.000 ns | 14.190 ns | 59.970 ns |
| Append to reused output | 2387.000 ns | 2.732 ns | 60.120 ns |
| Sort 4,096 handles | 236.557 ms | 0.668 ms | 3.131 ms |
| Resident format bytes | 69,408 | 1,986,088 | 115,720 |
| Serialized bytes | 65,269 | 1,985,705 | 111,007 |
| Source / serialized | 30.25× | 0.99× | 17.79× |

Concat compresses this fixture less densely than LZ4, but its direct comparisons
avoid decoding 64-KiB blocks. Raw reads/comparisons remain faster at a much larger
retained size. Comparisons and reused-buffer append allocate zero heap objects
in all three modes. Build allocations and every raw timing are included below.
There is no assertion that cold/warm SQL trade-offs can be inferred from this
microbenchmark; the focused SQL regression uses four total samples, warmup zero,
and CI owns the broader A/B workload and its unchanged regression gates.

Reproduction:

```sh
go test ./storage -run '^$' -bench '^BenchmarkStringCompression/(read|compare|append)' -benchtime=20000x -count=3 -benchmem
go test ./storage -run '^$' -bench '^BenchmarkStringCompression/(build|sort)$' -benchtime=3x -count=3 -benchmem
go test ./storage -run '^TestStringCompressionFootprint$' -v -count=1
```

### Retained samples

Values are `ns/op`; sample groups use the fixed iteration counts above.

| Probe | LZ4 samples | Raw samples | Concat samples |
|---|---|---|---|
| read-1 | 9.654, 7.814, 7.516 | 8.881, 7.375, 7.524 | 5.96, 4.841, 4.983 |
| read-0 | 9.975, 7.601, 7.573 | 8.327, 7.469, 7.452 | 4.881, 4.479, 4.448 |
| read-2 | 6.447, 6.372, 6.602 | 8.083, 7.372, 7.392 | 4.449, 5.482, 4.713 |
| build | 1.24994e+07, 1.20187e+07, 1.26757e+07 | 1.02379e+07, 1.09493e+07, 1.02102e+07 | 1.40134e+07, 1.35778e+07, 1.48248e+07 |
| compare | 4695, 4723, 4662 | 17.66, 10.35, 14.19 | 59.46, 60.58, 59.97 |
| append | 2387, 2375, 2439 | 2.732, 2.704, 2.733 | 59.31, 60.34, 60.12 |
| sort | 2.39935e+08, 2.36557e+08, 2.35548e+08 | 654976, 667753, 679946 | 3.13071e+06, 3.23314e+06, 3.01162e+06 |

Build allocation samples (`bytes/op`, `allocs/op`):

- lz4: (13276896, 96), (13273253, 92), (13273221, 92)
- raw: (12802309, 69), (12804168, 71), (12804168, 71)
- concat: (14597634, 7748), (14595733, 7745), (14593901, 7742)

## Focused SQL A/B after operator integration

Baseline `bdaa313a6`; identical checked-in fixtures and existing sample counts,
sequential runs on the same CPU with native Go and the same experimental JIT Go.
Values are milliseconds. Unicode uses all 501 cold/reuse queries, temporal uses
all 11 queries per fixture, and WordPress keeps its existing 10 warmups / 30 samples.
Native temporal uses the median of three predeclared fresh fixtures on each side;
all other rows use one fresh fixture per revision/toolchain. No forced GC, changed
query limits or accepted-regression waivers. Every sample, including the first,
is retained in [string-concat-sql-samples.json](string-concat-sql-samples.json).

| Workload | Native master | Native concat | Change | JIT master | JIT concat | Change |
|---|---:|---:|---:|---:|---:|---:|
| Unicode range, 501 executions (11 + 490 grouped) | 1041.2 | 986.9 | -5.2% | 813.8 | 842.7 | +3.6% |
| Temporal page, 11 executions | 27.8 | 31.2 | +12.1% | 23.3 | 22.4 | -4.1% |
| WordPress rare search | 4.3 | 4.0 | -7.2% | 4.7 | 3.1 | -33.8% |
| WordPress absent search | 3.9 | 4.0 | +2.5% | 3.1 | 3.1 | +0.1% |
| WordPress disjoint OR search | 3.5 | 3.7 | +4.7% | 3.1 | 3.0 | -3.4% |

Native temporal totals (all three retained): master 27.8 / 29.7 / 26.0 ms,
concat 71.5 / 31.0 / 31.2 ms. The isolated slow fixture is visible in the samples;
none of its query samples is removed from its total.

The three complete focused suites pass their result checks. CI still owns the
seven-fresh-fixture A/B verdict; these manual measurements do not replace it.
Reproduction: set `PERF_AB_MODE=record PERF_FIXTURE_TRIAL=1 PERF_BASELINE_FILE=OUTPUT.json`
and run `python3 run_sql_tests.py` with, one at a time,
`tests/performance/collation-comparison.yaml`,
`tests/performance/wordpress-search.yaml`, and
`tests/performance/temporal-order-output.yaml` in each revision's worktree.

### Final cast inlining probe

The broader SQL table above was captured at `38ab4857b`. Final cast integration
restores the baseline inlineable `Scmer.String` wrapper (compiler cost 174 → 67),
keeps one-span casts zero-copy in the append helper, and isolates concat numeric
scratch from ordinary coercion frames. No emitter is edited by hand.

The remaining CI cold-temporal regression was checked with three predeclared
fresh A/B pairs, alternating master then concat, using JIT and `GOMAXPROCS=4`.
Every fixture includes its first query plus ten reuses and unchanged results.

| Temporal workload, milliseconds | Master | Final concat | Change |
|---|---:|---:|---:|
| Median of three complete 11-query fixtures | 20.585 | 23.158 | +12.5% |
| Fixture 1 | 20.585 | 23.158 | |
| Fixture 2 | 20.526 | 23.442 | |
| Fixture 3 | 21.883 | 22.294 | |

All samples, including contemporaneous unpaired diagnostic runs under changing
host load, are retained in the JSON artifact. CI remains the authoritative
seven-fixture performance gate; no threshold or assertion is relaxed.


### Generated storage-reader fusion

The scalar reader no longer calls a `readValue` wrapper. Branching Scmer readers
and their constructors are lowered from their Go SSA by `make jitgen`, including
concat offsets, NULL paths and the pool handle. Scalar search helpers keep the
existing leaf-helper inlining policy. The generated emitters are not hand edited.

The generator now preserves scalar row-ID homes across sibling CFGs, distinguishes
Go string pairs from slice triples, and keeps operands live while allocating
slice-length and aggregate-field results. Descriptor-owned constructors no longer
look like variadic arguments. Pointer-conversion aliases share the canonical
lifetime. Fusion also exposed a filter compaction bug: a computed slice address
can be stack-backed under pressure and must be materialized before using its
register. The regression checks selected record IDs as well as their count.

The fused storage-to-filter test covers NULL, empty and fragmented Unicode
strings, `strlen(concat ...)`, no-op lowercase plus LIKE, and general-ci comparison.
It verifies identical results and zero allocations for all three predicates.

The existing CI A/B runner compared PR revision `891029576` with the initial reader-fusion fixes, before
pool-capacity reservation, using
one fresh prepared fixture per role. The fixture contains 22,000 temporal rows;
no samples, result assertions or timing limits were changed. Times are milliseconds.

| Workload | Previous PR JIT | Reader fusion fixes | Change |
|---|---:|---:|---:|
| First temporal page plus ten repetitions (total) | 24.199 | 22.705 | -6.2% |
| Warm temporal page (20 warmups, 100 samples, median) | 0.597 | 0.587 | -1.6% |
| Mixed decimal/temporal UNION (31 executions, total) | 40.165 | 41.694 | +3.8% |

These are development measurements against the previous PR, not a replacement
for CI's comparison with master. Full results and binary identities are retained
in `string-concat-sql-samples.json` under `jit_fusion_fix`.


### Cold-load headroom and retained repetitions

Legacy normalization no longer compiles three unused standalone readers for its
private offset directory. That directory is consumed through `GetValueUInt` or
its SSA in the parent. The builder reserves a small pool capacity from entry count,
uses smaller stack scratch with ordinary append on overflow, and avoids a duplicate
trigram-map lookup. The format and compression ratio are unchanged.

| 22,000 short strings, complete pool build | Previous PR | Reserved pool | Change |
|---|---:|---:|---:|
| Median time of three fixed 50-build probes | 4.672 ms | 3.776 ms | -19.2% |
| Temporary allocation per build | 3,557,364 B | 2,723,084 B | -23.5% |

All three probe results are retained, including the slower candidate probe. The
4096-row trace fixture still occupies 115,720 bytes resident and 111,007 bytes on
disk for 1,974,272 source bytes (17.785x).

Seven predeclared invocations of the unchanged temporal CI runner retained their
initial pairs and its automatic diagnostic repetitions: 37 fresh fixtures per
role. The complete cold total contains the first query and ten reuses; warm and
UNION counts stay at 100 samples/20 warmups and 31 total queries, respectively.

| All 37 fresh fixtures, median per role | LZ4 baseline | Final concat | Change |
|---|---:|---:|---:|
| First temporal page plus ten repetitions | 24.506 ms | 28.845 ms | +17.7% |
| Warm temporal page | 0.804 ms | 0.886 ms | +10.2% |
| Mixed decimal/temporal UNION | 54.082 ms | 51.484 ms | -4.8% |

Host timings vary substantially: two individual seven-fixture batches still
failed the cold gate (+45.9% and +26.4%); one also failed UNION (+25.5%). They are
not excluded or waived. The aggregate above is a development measurement; required
CI still applies its unchanged 20% gate to the latest commit against master.
Every batch result and every raw fixture sample is included in the JSON artifact.

Raw-format test fixtures now build their private payload before finalization,
rather than recoding a finished column behind its immutable specialized readers.
Production-finished concat columns retain direct JIT, bulk and fused-filter checks.

### Typed integer accessor after CI profile analysis

Revision `b077eb262` passes every functional, cross-JIT, upgrade, cluster and
packaging check. Its formerly failing temporal JIT workload passes seven fresh
fixtures at 33.103 → 38.694 ms (+16.9%); warm and UNION improve 1.3% and 2.3%.
Two different cases failed: native computed delta keys (+30.5%) and the JIT
single dynamic equality prefix (+38.8%). The native automatic profiled diagnostic
measured +3.6%; this does not replace the original failed verdict.

The ordered-prefix profile spends about 9–11% of CPU in `Scmer.Int`, on both
revisions. Its disassembly routed already typed integers through `GetTag` and
a jump table. `Int` now returns the sentinel-backed integer payload directly
through an inlineable wrapper; the full coercion switch remains for other types.
`Less` no longer calls `Int` on its typed integer path. `make jitgen` was rerun;
it produces no further emitter changes for this source change.

| Fixed 10,000,000-access probe, median of three | Before | After | Change |
|---|---:|---:|---:|
| Typed integer access | 2.602 ns | 0.424 ns | -83.7% |

Three predeclared unchanged CI-runner invocations per affected SQL workload
retain nine fresh fixtures per role, including automatic verification pairs.

| Complete cold workload, pooled median | LZ4 baseline | Integer accessor | Change |
|---|---:|---:|---:|
| Native computed delta keys, 11 queries | 36.330 ms | 44.668 ms | +23.0% |
| JIT one dynamic equality prefix, 10 queries | 154.217 ms | 118.602 ms | -23.1% |
| JIT three prefixes with residual acceptance, 10 queries | 385.637 ms | 322.084 ms | -16.5% |
| JIT forty prefixes, 10 queries | 113.808 ms | 117.835 ms | +3.5% |
| JIT broad keyset, 10 queries | 115.961 ms | 113.906 ms | -1.8% |
| JIT skewed hot prefix, 10 queries | 394.647 ms | 372.819 ms | -5.5% |
| JIT skewed rare prefix, 10 queries | 230.417 ms | 218.210 ms | -5.3% |

All three JIT batches pass. Native delta batches measure +37.4% (failed), -11.1%
and -5.9%; the first contains seven verification pairs and the others one each.
The aggregate native result is still above 20%, so it is explicitly unresolved
pending CI. Seven merged CI diagnostic profiles show 790 ms sampled CPU on the
baseline and 680 ms on the candidate, principally interpreter, environment maps,
parsing and B-tree construction; they do not establish increased CPU work or
identify a new concat allocation hot spot. No gate is waived or weakened.
All raw fixtures and the negative before/after results are preserved under
`integer_accessor_followup` in `string-concat-sql-samples.json`.

## Adaptive long-text placement (November 2026)

The development baseline is the pulled and rebuilt main checkout at
`762a2cf1a95923914249b14e1174e647df9cf592`. One feature worktree contains the
candidate. This changes compression choice, not the persisted wire format:
existing inline strings and blob references already support these lengths.
Historical readers, escaping, NULLs, and external blob ownership remain intact.

During the ordinary analysis scan, columns with more than eight distinct values
collect a deterministic reservoir of eight strings between 2 and 32 KiB.
Owned sample text is capped at 256 KiB, replacement copying at 1 MiB per column.
Values over 32 KiB stay blobs. Exact duplicate samples retain gzip, since whole
blob deduplication already shares them. A fixed 32-KiB fingerprint table tests
shared 16-byte windows; at least half the sample documents must have 75% shared
coverage. Sliding lookups tolerate shifted prefixes. Common PDF headers alone
are insufficient. A fingerprint hit only enables a real cost probe, so hash
collisions cannot change decoded content.

The subsequent concat probe limits its index to 8,192 trigrams and four offsets
per trigram. It reuses one gzip writer and counts output without retaining it.
The estimate charges the first pool seed plus the **worst** observed incremental
bytes per source byte, directory bytes and gzip reference overhead. A 20% space
margin is required. At most 2 MiB of additional source text is then inlined per
column build; excess rows keep gzip. A private row bitmap keeps scan/build
choices identical. Sample, bitmap and codec indexes are discarded before
publication. This is a bounded heuristic, not a guarantee of optimal size or
query speed for every unsampled row. In particular, large or heterogeneous
shared document families can be false negatives.

### Fixed manual A/B measurements

Native Go 1.26, AMD Ryzen 9 7900X3D, `GOMAXPROCS=4`. Identical 128-row synthetic
fixtures, including first execution and without forced GC. Build uses five
iterations per sample and five samples; operators use 10,000 iterations and
three samples; bigrams use 30 iterations and five samples. Targeted longer
unchanged-path controls use 100 build iterations for duplicates and 50 for PDF
shards 7/9, each with five samples. All earlier short/noisy and rejected-policy
measurements are retained in `string-concat-sql-samples.json` under
`adaptive_blob`. Table entries are medians, not end-to-end workload gates.

| Fixture | Master build | Adaptive build | Change | Master serialized | Adaptive serialized |
|---|---:|---:|---:|---:|---:|
| Shared high-entropy paragraph, distinct document prefixes | 36.197 ms | 10.331 ms | −71.5% | 655,185 B | 8,020 B |
| Shared repetitive template, distinct prefixes | 22.774 ms | 7.113 ms | −68.8% | 23,412 B | 9,228 B |
| Unrelated random text, unchanged gzip | 50.598 ms | 31.562 ms | −37.6%* | 652,002 B | 652,002 B |
| Identical whole documents, unchanged deduplication (longer probe) | 1.491 ms | 1.579 ms | +5.9% | 5,271 B | 5,271 B |
| Real PDF shard 7, unchanged gzip (longer probe) | 13.773 ms | 15.137 ms | +9.9% | unchanged | unchanged |
| Real PDF shard 9, unchanged gzip (longer probe) | 27.334 ms | 30.573 ms | +11.9% | unchanged | unchanged |

*The unchanged representation cannot explain the unrelated-text timing gain;
process scheduling/GC noise is visible in the retained samples. Do not attribute
this number to a new gzip optimization. The short duplicate probe initially
reported +26.7%; the predeclared longer identical A/B probe above resolves it.

For the shared high-entropy fixture, build allocation bytes fall from 109.67 MB
to 10.89 MB (−90.1%). Stored resident bytes fall from 650,736 to 9,112, while a
warm gzip cache additionally retains 770,816 decoded text bytes. The concat
column has no decoded cache. Counts exclude the source fixture, map overhead and
OS page cache. Serialized size includes detached gzip contents, so filesystem
block rounding is not credited to concat.

| Operation, shared high-entropy fixture | Master | Adaptive | Allocation consequence |
|---|---:|---:|---|
| Uncached row access | 49,263 ns | 28.67 ns | gzip decode disappears |
| Warm string materialization | 6.982 ns | 935.6 ns | fragmented concat allocates its result |
| Warm append to reused output | 3.510 ns | 82.54 ns | both allocation-free |
| Byte length | 4.973 ns | 6.664 ns | both allocation-free |
| Equality | 7.899 ns | 81.28 ns | both allocation-free |
| Binary less | 9.775 ns | 56.93 ns | both allocation-free |
| General-ci less | 60.70 ns | 105.7 ns | both allocation-free |
| Hash | 383.8 ns | 613.8 ns | both allocation-free |
| LIKE `%wartung%`, general-ci | 3,541 ns | 4,839 ns | cursor traversal versus cached text |
| Bigram build, cold | 11.488 ms | 8.930 ms | reduced gzip allocation |
| Bigram build, warm | 5.447 ms | 9.068 ms | +66.5% CPU despite less decoded RAM |

“Cold” uses the actual uncached blob reader and includes gzip decoding. “Warm”
uses retained, content-deduplicated decoded blob handles versus concat handles;
it excludes initial decoding and separately counts its retained RAM. This is
an operator comparison, not a simulation of production blob-cache admission.
The repetitive-template bigram probe also loses: cold 5.432 → 9.412 ms (+73.3%),
warm 4.465 → 8.964 ms (+100.8%). Concat therefore does **not** universally improve
bigram construction. Warm native string operations can be substantially faster
than cursor operations, paid for with a decoded cache. Query workload matters.

### Actual PDF extracts and rejected choices

The private corpus contains 1,102 distinct UTF-8 PDF extracts (13,565,905 bytes)
from 12 actual `fop_files` shard groups. Values are sampled only within their
original shard, using seed 1729, and restricted to this policy's 2–32 KiB window.
Corpus SHA-256:
`c86a9488546d5f2412fa32e1ff2b73535c68286333517a1cb68f6ed4a092bb61`.
Document contents and individual hashes are omitted from the repository.

An initial mean-growth estimator selected two shards: serialized bytes changed
299,733 → 259,656 (−13.4%) and 63,643 → 126,674 (+99.0%). That false positive
motivated the worst-growth estimate and stronger fingerprint gate. The final
heuristic retains gzip on **all 12** groups: no measured PDF storage saving and
no serialized growth. It does not infer a benefit from the synthetic examples.
The longer final bigram controls on unchanged PDF shards are approximately
unchanged: shard 0 cold −1.8%/warm +2.2%, shard 9 +1.5%/+2.3%, shard 11
+3.7%/+5.1%. Earlier three-iteration controls varied widely and remain available.

To reproduce the public synthetic measurements in each checkout, install the
identical `BenchmarkAdaptiveBlob` instrumentation, build the test binary, and run
these fixed probes sequentially on an otherwise idle machine:

```sh
GOMAXPROCS=4 go test -tags=nophp ./storage -run '^$' -bench '^BenchmarkAdaptiveBlob/.*/pipeline/build$' -benchtime=5x -count=5 -benchmem
GOMAXPROCS=4 go test -tags=nophp ./storage -run '^$' -bench '^BenchmarkAdaptiveBlob/.*/pipeline/(read-cold|materialize-warm|append-warm|strlen-warm|equal-warm|less-warm|less-ci-warm|hash-warm|like-warm)$' -benchtime=10000x -count=3 -benchmem
GOMAXPROCS=4 go test -tags=nophp ./storage -run '^$' -bench '^BenchmarkAdaptiveBlob/.*/pipeline/bigram-(cold|warm)$' -benchtime=30x -count=5 -benchmem
```

`MEMCP_BLOB_CORPUS` optionally supplies a local JSON array of
`{"name": "shard-group", "texts": ["..."]}` without changing the public fixtures.
Use the same input, compiler and counts on both revisions. Neither the CI
regression budget nor an existing assertion is relaxed by this feature.

The complete focused SQL string-compression suite passes 88/88 on both builds.
Subsequent build-info verification found master was Go 1.26 without JIT, while
the candidate used Go JIT 1.27.3 (nophp). The original compiler label was wrong.
The new post-restart workload checks 512 long escaped documents, a mix of inline
text and blob fallback, with four total LIKE queries and zero warmup:
167.952 → 126.999 ms, but this is **not a valid JIT A/B comparison** and the
claimed speedup is withdrawn. All QUERY_TIME records, including other queries
that became slower in this individual run, are retained in the JSON. These are
one suite run per side, not a replacement for CI A/B. Focused native, race,
JIT scalar/range/multi and fused-filter checks also pass; emitters are generated
by `make jitgen`. Upgrade fixtures add 150 long documents without a format bump.

### CI failure investigation and parser root ordering

The original head `b547a1045` had four red checks: a dependency-download failure,
two JIT performance shards and their aggregate. Actual seven-fixture failures
were nested scalar sums 135.489 → 163.330 ms (+20.5%; 0.744 ms above the 20%
limit) and a filtered scalar binder 166.291 → 203.379 ms (+22.3%; 3.830 ms above
the limit). Subsequent profiled diagnostics also received SIGSEGV, including
in the baseline binary. Native performance, correctness and upgrade CI passed.

An independent CI repeat on the **same source commit**, with no assertion or
budget changes, passed every check: nested sums 204.316 → 207.993 ms (+1.8%,
seven fixtures), filtered scalar binder 189.630 → 188.672 ms (−0.5%, one fixture),
negated snapshots 58.259 → 58.549 ms (+0.5%, seven fixtures). The original failing
measurements remain in the JSON; a passing repeat does not make them disappear.

The correctly rebuilt local JIT/PHP comparison also changes which case loses:
nested sums 208.886 → 189.434 ms (−9.3%), filtered scalar binder
237.711 → 237.123 ms (−0.2%), but negated snapshots 39.950 → 54.338 ms (+36.0%),
each across seven fresh fixtures with the unchanged CI configuration. Four
separate GC diagnostic pairs show overlapping concurrent GC on some slow calls;
they do not prove that GC explains every difference. No GC is forced or disabled.

For nested sums and negated snapshots, EXPLAIN, IR, PHYSICAL and REORDER outputs
match byte for byte across the diagnostic processes. JIT import statistics
(8,239 records per side) and non-parser named code sizes match (8,592 named
functions per side). One 1,761-byte storage reader has 406 identical instructions
after normalizing absolute addresses. There is no observed loss of operator
fusion in that comparison.

Native **parser grammar** code sizes do differ. Environment roots were compiled
in Go map iteration order, changing rule IDs, memo layout and generated code.
The correction sorts eligible binding names before alias deduplication, retaining
whitespace grouping and the original order of alternatives inside each parser.
A focused test reproduces unstable IDs before the fix and verifies stable IDs,
code length, shared grammar publication and results over 16 independent builds
afterwards. All focused JIT parser tests pass. This closes a concrete source of
layout variation; it is not proof that parser order caused all CI fluctuations.

With stable parser root ordering, the unchanged focused A/B configuration passes
both affected suites: negated snapshots 53.782 → 50.057 ms (−6.9%, seven
fixtures), nested scalar sums 172.333 → 169.985 ms (−1.4%, seven fixtures),
and filtered scalar binder 256.880 → 247.785 ms (−3.5%, one fixture). All cases
in each suite pass. These are local measurements against `762a2cf1a`, before
rebasing onto `9cb71ca8b` (the latter adds a guard against loading unused catalogs
during global cleanup). No workload or regression budget is changed.

After rebasing, both main and feature checkouts were explicitly built and their
build info verified as `go1.27.3-X:jit`, with identical PHP tags. The complete
focused SQL suite passes **88/88 on each side**. Its four-query post-restart
adaptive case measures **103.161 → 93.473 ms (−9.4%)**. This replaces the withdrawn
mixed-compiler SQL comparison. Binary hashes and every QUERY_TIME record are
retained under `adaptive_blob.focused_sql_verified_jit`; no speedup from one suite
run is substituted for the CI A/B gate.
