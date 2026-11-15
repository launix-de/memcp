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
