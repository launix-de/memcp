# Exact compressed strings and additional formats

Copyright (C) 2026 Carl-Philip Hänsch. Licensed under GPL-3.0-or-later.

Base64 already uses BString: its payload is decoded binary data, while the
value still represents the original Base64 text. No additional runtime tag is
needed. Prefix compression remains the responsibility of StoragePrefix.

## Correctness and compatibility

- Format detection now requires canonical Base64, including padding shape and
  unused tail bits. `Zh==`, `Zm9=`, and `AA=A` remain exact strings instead of
  being normalized or causing a rebuild panic. Unpadded input receives the same
  strict checks.
- Empty strings disqualify fixed-width UUID encoding, fixing rebuilds containing
  UUIDs and empty strings (including all-empty columns).
- Binary equality preserves alphabet and padding distinctions. SQL equality
  keeps the existing text case-folding semantics; explicit binary collations
  no longer accidentally use SQL case-insensitive equality for compressed values.
- Existing format IDs and readers remain unchanged. StorageString version 3
  introduces permanent IDs 17–20. Readers for legacy and versions 0, 1, and 2
  remain available. Existing files require no migration; a future rebuild can
  choose the new formats. Older executables cannot read version 3 columns.

| ID | Representation | Payload |
| --- | --- | --- |
| 17 | Unpadded standard Base64 | Decoded bytes |
| 18 | Unpadded URL Base64 | Decoded bytes |
| 19 | Timestamp alphabet ` -.0123456789:TZ` | Ordered high-first nibbles |
| 20 | Timestamp alphabet `+-.0123456789:TZ` | Ordered high-first nibbles |

Timestamp formats are lossless alphabet encodings, not date parsers. They
preserve spelling and fractional precision. Mixed alphabets fall back to another
eligible format or raw strings. Both dictionary and non-dictionary storage,
nullable columns, bulk readers, serialization, and reload support the new IDs.

## Operators

BString carries URL and unpadded flags in transient metadata. Shared bounded
string views extend comparisons, LIKE, substrings, length, trimming, case
conversion, concatenation, hashes, output, and Base64/hex operations to the new
formats. Base64 decoding retains the public decoder's rejection of unsupported
padding/alphabet variants.

Binary BString equality with identical encoding flags compares stored bytes.
Long mixed binary equality strictly decodes the other operand in 256-character
blocks into a 192-byte stack buffer, compares raw bytes, and stops on a decisive
mismatch. Short mixed comparisons retain the faster fused loop. Aligned CString
equality compares complete packed bytes and masks only boundary nibbles.

Base64 sorting must compare encoded text: raw byte order differs from Base64
alphabet order. Same-format BString sorting skips equal raw blocks and decodes
only the sextets affected by the first differing byte. Mixed sorting and SQL
case-insensitive equality stop at the first decisive block. Short values use a
16-bit plus 8-bit load, four constant sextet extractions, and a 32-bit text
comparison; final partial groups stay outside the loop. Longer values use the
chunk encoder. All these comparison paths avoid full-string allocations.

## Manual measurements

Measurements compare development baseline `2c3f45274` (master including PR #846)
with this branch. The detached benchmark baseline uses `1d7cd1bfd`, whose source
matches that master commit except README. Both use identical benchmark fixtures.
Host: AMD Ryzen 9 7900X3D, linux/amd64. Other workloads were active, so small
changes should be treated as noise rather than guaranteed improvements.

Results below are manual development measurements, independent of CI.

### Loop experiment

Fixed CPU 2, GOMAXPROCS=1, 500 ms per case, ABBA order, two samples per variant:
short (16 raw bytes) late-difference SQL equality improved from 48.01 to 32.355 ns
(-32.6%) with fused sextet processing. At 2048 raw bytes, late-difference sorting
regressed from 1939 to 2178 ns (+12.3%), and SQL equality from 1487 to 1735 ns
(+16.7%). Therefore only values up to 64 text bytes use the fused loop.

The additional strict-decoding equality experiment (same setup) measured long
late-difference binary equality at 1722.5 versus 1551 ns (-10.0%) compared with
the all-fused prototype. Short values regressed, so they retain fused comparison.
These intermediate experiments are separate from the final master comparison.

### Practical limits

Bulk column readers still materialize text into a shared arena before many SQL
operators. Therefore direct compressed-operator microbenchmarks do not predict
the same speedup for a full SQL projection. Repeated SQL ORDER BY measurements
can reuse an already-built index; the separate 4096-value sorting benchmark
measures actual comparison work. General language-collation sorting still has
materialization fallbacks. Dictionary payload sizes exclude column metadata,
row indexes, and outer compression.


### Plan inspection

EXPLAIN, EXPLAIN IR, EXPLAIN PHYSICAL, and EXPLAIN REORDER were inspected for
all three new SQL performance cases. Both projections lower to a single `scan`
with their scalar operators in the result-row callback; ORDER BY LIMIT lowers
to `scan_order` with a 100-row limit. No planner changes are required here.

### Validation

169 targeted SQL cases pass: string compression (70), scalar functions (51),
string functions (29), and LIKE (19). The compression suite includes rebuild and
restart. Targeted Go tests include old on-disk CString fixtures, all Base64
representation pairs and bit positions, new dictionary/non-dictionary formats,
nullable/bulk reads, operator parity, and UUID/empty-string regressions.


### Final operator and build A/B

Stock Go 1.24.0, CPU 2, GOMAXPROCS=1; ABBA, two samples/version,
200 ms/case with Go's iteration calibration. Tables show arithmetic means.
No additional explicit warmup is used for these Go benchmarks. `B` denotes
BString, `C` CString, `S` plain string. Differences are text-byte positions.
All candidate Base64 comparisons below use zero bytes and zero allocations.

The shared host was heavily loaded in this sweep. Even repeated instances of
the same unchanged same-value comparison varied substantially. These are
observations, not confidence intervals; the focused CString control below
resolves the apparent large CString regressions in this noisy sweep.

| Benchmark | Baseline ns/op | Candidate ns/op | Change |
| --- | ---: | ---: | ---: |
| `CStringCompare/difference-0/equal-CS` | 41.45 | 40.67 | -1.9% |
| `CStringCompare/difference-0/equalSQL-CS` | 36.47 | 53.82 | +47.6% |
| `CStringCompare/difference-0/less-CS` | 35.41 | 49.52 | +39.8% |
| `CStringCompare/difference-0/less-CC` | 35.36 | 55.38 | +56.6% |
| `CStringCompare/difference-31/equal-CS` | 53.73 | 69.60 | +29.5% |
| `CStringCompare/difference-31/equalSQL-CS` | 70.16 | 83.66 | +19.2% |
| `CStringCompare/difference-31/less-CS` | 39.50 | 63.02 | +59.5% |
| `CStringCompare/difference-31/less-CC` | 34.51 | 37.67 | +9.2% |
| `CStringCompare/difference-63/equal-CS` | 61.11 | 90.62 | +48.3% |
| `CStringCompare/difference-63/equalSQL-CS` | 95.75 | 128.20 | +33.9% |
| `CStringCompare/difference-63/less-CS` | 82.81 | 89.53 | +8.1% |
| `CStringCompare/difference-63/less-CC` | 34.02 | 30.79 | -9.5% |
| `CStringCompare/difference-64/equal-CS` | 84.00 | 79.55 | -5.3% |
| `CStringCompare/difference-64/equalSQL-CS` | 101.84 | 106.99 | +5.1% |
| `CStringCompare/difference-64/less-CS` | 85.44 | 98.87 | +15.7% |
| `CStringCompare/difference-64/less-CC` | 37.56 | 31.77 | -15.4% |
| `CStringCompare/uuid-less-CC` | 38.12 | 35.20 | -7.7% |
| `CStringIndexSort` | 2371475.50 | 2268547.00 | -4.3% |
| `Base64Compare/16/difference-0/equal-BS` | 100.70 | 19.61 | -80.5% |
| `Base64Compare/16/difference-0/less-BS` | 104.29 | 11.13 | -89.3% |
| `Base64Compare/16/difference-0/less-BB` | 229.50 | 41.18 | -82.1% |
| `Base64Compare/16/difference-0/equal-BB` | 12.62 | 10.28 | -18.6% |
| `Base64Compare/16/difference-0/equal-same-BB` | 9.72 | 9.19 | -5.5% |
| `Base64Compare/16/difference-0/equalSQL-BS` | 83.22 | 17.94 | -78.4% |
| `Base64Compare/16/difference-0/equalSQL-same-BB` | 8.70 | 9.90 | +13.7% |
| `Base64Compare/16/difference-20/equal-BS` | 95.64 | 46.09 | -51.8% |
| `Base64Compare/16/difference-20/less-BS` | 91.37 | 54.42 | -40.4% |
| `Base64Compare/16/difference-20/less-BB` | 120.25 | 66.07 | -45.1% |
| `Base64Compare/16/difference-20/equal-BB` | 9.90 | 14.09 | +42.4% |
| `Base64Compare/16/difference-20/equal-same-BB` | 6.99 | 11.35 | +62.3% |
| `Base64Compare/16/difference-20/equalSQL-BS` | 99.37 | 57.87 | -41.8% |
| `Base64Compare/16/difference-20/equalSQL-same-BB` | 7.60 | 11.65 | +53.2% |
| `Base64Compare/2048/difference-0/equal-BS` | 3010.00 | 34.05 | -98.9% |
| `Base64Compare/2048/difference-0/less-BS` | 3097.00 | 15.06 | -99.5% |
| `Base64Compare/2048/difference-0/less-BB` | 5953.00 | 42.88 | -99.3% |
| `Base64Compare/2048/difference-0/equal-BB` | 9.76 | 10.68 | +9.4% |
| `Base64Compare/2048/difference-0/equal-same-BB` | 7.10 | 11.41 | +60.7% |
| `Base64Compare/2048/difference-0/equalSQL-BS` | 2734.00 | 15.07 | -99.4% |
| `Base64Compare/2048/difference-0/equalSQL-same-BB` | 7.83 | 9.36 | +19.5% |
| `Base64Compare/2048/difference-2728/equal-BS` | 2862.50 | 2414.00 | -15.7% |
| `Base64Compare/2048/difference-2728/less-BS` | 3718.50 | 3007.50 | -19.1% |
| `Base64Compare/2048/difference-2728/less-BB` | 4951.00 | 496.45 | -90.0% |
| `Base64Compare/2048/difference-2728/equal-BB` | 30.54 | 33.84 | +10.8% |
| `Base64Compare/2048/difference-2728/equal-same-BB` | 9.08 | 7.81 | -13.9% |
| `Base64Compare/2048/difference-2728/equalSQL-BS` | 3996.50 | 2277.50 | -43.0% |
| `Base64Compare/2048/difference-2728/equalSQL-same-BB` | 9.12 | 10.65 | +16.7% |
| `Base64IndexSort` | 12487670.00 | 3602135.50 | -71.2% |

Focused CString control: CPU 9, GOMAXPROCS=1, 1 second/case, ABBA,
two samples/version, same fixtures and Go iteration calibration:

| Benchmark | Baseline ns/op | Candidate ns/op | Change |
| --- | ---: | ---: | ---: |
| `CStringCompare/difference-0/equal-CS` | 25.020 | 23.285 | -6.9% |
| `CStringCompare/difference-0/equalSQL-CS` | 22.890 | 24.035 | +5.0% |
| `CStringCompare/difference-0/less-CS` | 21.695 | 21.720 | +0.1% |
| `CStringCompare/difference-63/equal-CS` | 56.855 | 49.865 | -12.3% |
| `CStringCompare/difference-63/equalSQL-CS` | 88.335 | 68.415 | -22.6% |
| `CStringCompare/difference-63/less-CS` | 59.910 | 48.190 | -19.6% |

Build fixture: 128 rows with 8 distinct values; size is raw input bytes for
Base64, text length for hex. Timestamp values contain fractional seconds and
`+02:00`. This measures format selection plus column construction. Extra format
validation and actual decoding of newly compressible inputs can increase build
cost and allocations; the read and space benefits do not make rebuilding free.

| Format/size | Baseline µs | Candidate µs | Change | Dictionary bytes before → after |
| --- | ---: | ---: | ---: | ---: |
| hex/16 | 72.89 | 83.15 | +14.1% | 64 → 64 |
| hex/2048 | 27621.39 | 31526.70 | +14.1% | 8192 → 8192 |
| base64/16 | 104.14 | 126.49 | +21.5% | 128 → 128 |
| base64/2048 | 45672.67 | 44164.78 | -3.3% | 16384 → 16384 |
| raw-base64/16 | 94.20 | 124.36 | +32.0% | 176 → 128 |
| raw-base64/2048 | 43432.87 | 44922.30 | +3.4% | 21848 → 16384 |
| timestamp/16 | 103.54 | 119.99 | +15.9% | 232 → 116 |

The Base64 sorting benchmark sorts 4096 shuffled compressed values each
iteration. Allocation drops from roughly 13 MB / 204741 allocations per sort
to 88 bytes / 3 allocations (sort closure bookkeeping). CString sort is an
independent control with 4096 values.

Targeted race tests and JIT compressed-operator/representation tests also pass.


### SQL A/B

JIT-enabled Go toolchain at `84fe25ee`, GOMAXPROCS=4, CPUs 0–3; the same
4096-row fixtures from `tests/storage/formats/string-compression.yaml` on both
versions. The initial suite-level ABBA used 5 warmups and 31 samples/query,
with two runs/version. Its means of runner-reported timings appear below.
These runs suffered substantial host-load drift, including apparent regressions
in unchanged cached-index and materialized-string paths.

The follow-up kept two isolated, warmed servers alive and immediately alternated
requests A/B/B/A for 16 cycles: 32 samples/version/query after 5 warmups each.
Its values are median client wall time (including HTTP), not the runner's metric.
Do not compare absolute timings across these two methodologies. The interleaved
comparison removes the long delay between baseline and candidate samples.

| Query | Suite baseline ms | Suite candidate ms | Suite change | Interleaved baseline ms | Interleaved candidate ms | Interleaved change |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| CString mixed SQL equality projection | 14.650 | 18.750 | +28.0% | 4.810 | 4.893 | +1.7% |
| CString LIKE projection | 18.350 | 20.950 | +14.2% | 5.352 | 5.406 | +1.0% |
| CString ordered range | 2.600 | 3.000 | +15.4% | 0.986 | 0.986 | +0.0% |
| Compressed text composed projection | 23.400 | 22.750 | -2.8% | 7.187 | 7.266 | +1.1% |
| Compressed text Base64 roundtrip projection | 19.050 | 19.350 | +1.6% | 5.156 | 5.114 | -0.8% |
| Base64 mixed early comparison projection | 21.300 | 20.900 | -1.9% | 8.388 | 8.342 | -0.5% |
| Base64 index sort | 1.400 | 1.950 | +39.3% | 0.495 | 0.479 | -3.3% |
| Timestamp compressed operator projection | 27.200 | 27.150 | -0.2% | 8.737 | 8.953 | +2.5% |

Reproduce the operator sweep by building the same benchmark file on baseline
and candidate, then running each test binary from its `storage` directory:

```sh
GOMAXPROCS=1 taskset -c 2 ./storage.test -test.run='^$' \
  -test.bench='Benchmark(CStringCompare|CStringIndexSort|Base64Compare|Base64IndexSort|StringFormatBuild)$' \
  -test.benchtime=200ms -test.count=1
```

For SQL, create the two performance fixtures from the YAML on separate fresh
instances, rebuild, then execute its eight performance queries using the
warmup/sample configurations above. Exclude setup, rebuild, startup and reload
from query timings. The SQL fixtures explicitly insert 4096 rows; the runner's
calibrated `rows` metadata does not change these hardcoded fixture sizes.
