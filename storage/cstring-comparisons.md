<!-- Copyright (C) 2026 Carl-Philip Hänsch; SPDX-License-Identifier: GPL-3.0-or-later -->

# CString comparison and ordered nibble formats

CString comparisons read compressed characters until the result is known.
`Equal`, `EqualSQL`, `Less`, binary collations and the existing `general_ci`
ordering support compressed/plain operands in either order and compressed
operands with different formats or offsets. UUIDs of the same case compare
as their 16 raw bytes. Legacy nibble comparisons skip equal packed prefixes;
ordered nibble comparisons use byte ordering for aligned interior spans and
mask neighbouring entries at the edges. Mixed comparisons decode two characters
per byte using immutable lookup tables, with no decoded string allocation.

SQL equality applies its documented case-insensitive semantics to compressed
pairs as well as mixed/plain pairs. Previously compressed/compressed equality
could incorrectly distinguish uppercase and lowercase values. Unicode folding
and language-specific collators retain the existing canonical implementation
when the ASCII fast path cannot establish the result. The existing special
`general_ci` classification, including its leading `aa` rule and descending
ordering, is preserved.

`strlike` and `strlike_cs` use direct prefix/suffix access and stop when the
answer is known. Literal contains patterns up to 128 bytes decode into a
256-byte stack buffer with `pattern length - 1` overlap. Longer literals and
general `%`/`_`/escape patterns use a matcher that retains the last wildcard
position; they never require a complete decoded value. ASCII search patterns
need no allocated lowercase copy. Unicode patterns retain canonical lowercase
semantics. `strlen` uses the encoded length; `substr` and `sql_substr` decode
only the requested interval, preserving their respective bounds behavior.

Bulk column reads still decode into their existing shared arena. These changes
principally benefit lazy values, including scalar readers and index keys; an
HTTP projection over already decoded batches cannot realize the same gains
as the isolated comparison kernel. The planner and cache policy are unchanged.

## Permanent format assignments and compatibility

| Alphabet | Legacy ID (low nibble first) | Ordered ID (high nibble first) |
|---|---:|---:|
| Hex lowercase | 2 | 11 |
| Hex uppercase | 3 | 12 |
| Phone | 1 | 13 |
| Phone/DTMF | 10 | 14 |
| Decimal | 8 | 15 |
| Date/time | 9 | 16 |

Old IDs, alphabets and nibble order remain permanent. New ordered alphabets
are sorted by byte value. Format selection writes the ordered IDs when a
column is built/rebuilt; loading an old column never transcodes it. Serializing
a loaded old column preserves its format ID and dictionary encoding.

StorageString retains magic byte 20. Version 2 declares the additional IDs;
its body is the V1 body, read through a new V2 helper. Both the V0 and V1 readers
remain present. The historical raw header is identified by its actual ASCII
`1` sentinel (49), rather than interpreting every ID greater than 10 as raw.
Unknown IDs and ordered IDs paired with an older version fail explicitly.
This guarantees new readers can read old data; it does not make old binaries
capable of reading V2 files.

The in-memory Scmer remains 16 bytes. Its transient CString payload now uses
five format bits, one offset bit and 42 length bits; Scmer aux words are not
part of the persisted StorageString format. No persistent deletion, eviction,
WAL, table locking or engine durability behavior changes.

## Validation and reproduction

`storage/testdata/cstring-legacy.json` contains V1 bytes produced by baseline
`ae5c20c5d`, together with V0/header variants of the unchanged uncompressed body.
Tests cover dictionary and buffer modes, LZ4 dictionaries, scalar/range/multi
reads, reserialization and rebuilding. Comparisons exercise all old/new nibble
alphabets, both offsets, differing offsets, prefixes, empty values, UUIDs,
Unicode fallbacks, case variants, wildcard boundaries and invalid substring
bounds. The SQL suite adds ordered punctuation, comparisons, failed writes,
substring/length and shutdown/reload checks.

Targeted commands (stock Go and the project's JIT Go toolchain):

```sh
go test ./storage ./scm -run 'Test(CString|Format|String|StorageString|StrLike|Nibble|CompressDictionary|Bulk|BinaryCollation)' -count=1
go test -race ./storage -run '^TestCString' -count=1
PERF_TEST=1 python3 run_sql_tests.py tests/storage/formats/string-compression.yaml 18526
```

Also exercise `tests/sql/expressions/strings-like.yaml`,
`tests/sql/expressions/string-functions.yaml`,
`tests/sql/expressions/collation-columns.yaml` and
`tests/planner/order-window/collations-order.yaml`.

`EXPLAIN`, `EXPLAIN IR`, `EXPLAIN PHYSICAL` and `EXPLAIN REORDER` were inspected
for the three SQL benchmark queries. They retain a single query block lowered
to direct scans: equality/LIKE are evaluated in the per-row projection, while
the ordered-range query uses `scan_order` with its bound and limit. There is no
aggregate result cache obscuring the measured work.

Validated locally: **146/146 JIT SQL cases**, targeted stock/JIT Go tests,
and the CString race tests pass. Full repository suites run in PR CI per the
performance workflow. Generated emitters were refreshed with `make jitgen`.

## Manual A/B measurements

Baseline: `ae5c20c5d`. AMD Ryzen 9 7900X3D, `GOMAXPROCS=4`, same host and fixtures.
Go 1.24 microbenchmarks use standard testing.B calibration followed by a 200 ms
measurement in each A/B/B/A block; values below average the two blocks per
revision. SQL uses the project's JIT toolchain, identical rebuilt 4,096-row
fixtures, five warmups and 21 measured requests per query per A/B/B/A block.
SQL numbers average the per-block median HTTP latencies.

Reproduce the kernels with:

```sh
GOMAXPROCS=4 go test ./storage -run '^$' -bench '^BenchmarkCString' -benchtime=200ms
```

For baseline comparison, copy only `cstring_comparison_bench_test.go` into a
baseline worktree. SQL fixtures and queries are in the performance cases of
`tests/storage/formats/string-compression.yaml`; use the same warmup/sample
configuration on both revisions. The isolated sort benchmark sorts 4,096
CString keys through `Less`; it is not an end-to-end index build benchmark.

| Kernel | Baseline ns/op | Change ns/op | Change |
|---|---:|---:|---:|
| Compare/difference-0/equal-CS | 66.75 | 20.98 | -68.6% |
| Compare/difference-0/equalSQL-CS | 67.38 | 21.88 | -67.5% |
| Compare/difference-0/less-CS | 66.63 | 20.09 | -69.8% |
| Compare/difference-0/less-CC | 120.15 | 20.46 | -83.0% |
| Compare/difference-31/equal-CS | 67.81 | 31.69 | -53.3% |
| Compare/difference-31/equalSQL-CS | 78.25 | 37.49 | -52.1% |
| Compare/difference-31/less-CS | 65.13 | 29.90 | -54.1% |
| Compare/difference-31/less-CC | 122.50 | 20.17 | -83.5% |
| Compare/difference-63/equal-CS | 64.99 | 43.60 | -32.9% |
| Compare/difference-63/equalSQL-CS | 90.06 | 56.22 | -37.6% |
| Compare/difference-63/less-CS | 69.95 | 41.30 | -41.0% |
| Compare/difference-63/less-CC | 122.70 | 20.73 | -83.1% |
| Compare/difference-64/equal-CS | 65.04 | 43.64 | -32.9% |
| Compare/difference-64/equalSQL-CS | 97.61 | 56.36 | -42.3% |
| Compare/difference-64/less-CS | 70.65 | 41.70 | -41.0% |
| Compare/difference-64/less-CC | 126.50 | 20.00 | -84.2% |
| Compare/uuid-less-CC | 67.18 | 22.38 | -66.7% |
| IndexSort | 4683214.00 | 1282956.00 | -72.6% |
| StringOps/prefix | 1555.00 | 49.91 | -96.8% |
| StringOps/suffix | 1728.50 | 51.33 | -97.0% |
| StringOps/contains | 1661.00 | 178.35 | -89.3% |
| StringOps/absent | 4582.50 | 1556.00 | -66.0% |
| StringOps/length | 1567.00 | 1.74 | -99.9% |
| StringOps/substring | 1486.50 | 32.87 | -97.8% |
| Legacy/difference-0/less-CC | 123.90 | 24.86 | -79.9% |
| Legacy/difference-0/equal-CS | 65.35 | 21.16 | -67.6% |
| Legacy/difference-64/less-CC | 124.20 | 27.12 | -78.2% |
| Legacy/difference-64/equal-CS | 67.86 | 41.70 | -38.6% |

All measured comparisons and LIKE operations allocate zero decoded bytes.
The sort drops from roughly 100,334 allocations / 3.2 MB per operation to
three sort-wrapper allocations / roughly 100 bytes. A 16-byte substring
allocates its 16-byte result instead of a complete 2,048-byte value.

| JIT SQL query | Baseline ms | Change ms | Change |
|---|---:|---:|---:|
| CString mixed SQL equality projection | 5.900 | 5.700 | -3.4% |
| CString LIKE projection | 6.700 | 6.400 | -4.5% |
| CString ordered range | 1.900 | 1.400 | -26.3% |

HTTP measurements on this shared host include transport, JSON output and scan
costs, and remain noisier than the kernel measurements. In particular, batch
projections already receive arena-decoded strings; no CString-specific SQL
speedup is inferred from such a projection alone.

## Follow-up: decode complete bytes in pairs

The general storage decoder now peels an initial partial byte, runs a loop
containing only complete two-character decodes, then handles a single trailing
character after the loop. Legacy and ordered nibble order are selected outside
the loop. This also benefits callers that materialize strings and the shared
bulk decode arena. The existing whole-byte comparison paths are unchanged.

A targeted comparison against the PR's initial `a4f82aa79` decoder used both
function bodies in the same Go 1.24 benchmark binary, preallocated input/output,
`GOMAXPROCS=4`, 50 ms calibration/measurement and two samples per case. Means
for an initial nibble offset of one:

| Characters | Format | Initial PR ns/op | Pair decoder ns/op | Change |
|---|---|---:|---:|---:|
| 3 | Ordered hex | 4.120 | 3.351 | −18.7% |
| 16 | Ordered hex | 14.960 | 7.500 | −49.9% |
| 64 | Legacy hex | 60.810 | 24.765 | −59.3% |
| 64 | Ordered hex | 57.460 | 22.135 | −61.5% |
| 1024 | Legacy hex | 835.650 | 337.900 | −59.6% |
| 1024 | Ordered hex | 815.300 | 322.000 | −60.5% |

`BenchmarkStringNibbleDecode` retains the lengths, offsets and formats for
future A/B comparisons. Empty output with a nil source is also tested.

A separate prototype replaced the mixed-offset comparison fallback with pair
lookups. Identical 64-character values improved from 104.35 to 66.61 ns, but an
immediate mismatch worsened from 4.36 to 5.15 ns; three-character cases also
regressed. That prototype was not adopted. These measurements isolate the
fallback itself, not the already optimized whole-byte paths or full queries.

After the decoder change, the 55-case compression SQL suite, targeted Go and
race checks pass again. The SQL table above was refreshed with a new A/B/B/A
run against the original development baseline; the kernel tables are unchanged
except for the separately reported decoder measurements.

## Follow-up: compressed text operators

The follow-up branch compares against `efb20afac` (merged PR #844), rather than the
original pre-CString-optimization baseline above. It extends 25 builtin names:
`string?`, `concat`, `sql_concat`, `substr`, `sql_substr`, `strlen`, `strlike`,
`strlike_cs`, `toLower`, `toUpper`, the six Scheme/SQL trim variants,
`fnv_hash`, `stable_structural_hash`, `md5`, `sha1`, `sha256`, `base64_encode`,
`base64_decode`, `bin2hex`, and `hex2bin`.

- Type checks use tags; the legacy `Any`-wrapped Go string case remains valid.
  Concatenation only checks `io.Reader` for `tagAny`, eliminating the previous
  double decode. Concrete builder/writer helpers keep chunk buffers on the stack.
- BString length uses the raw byte count and Base64 padding rules. Its operator
  view generates only requested Base64 groups, with separate partial groups at
  the edges. Packed CString comparisons never receive this Base64 view.
- Nibble CString substrings retain a GC-visible pointer and nibble offset into
  immutable source storage. BString and UUID slices decode only the requested
  text range. Full-range slices and unchanged trims reuse their input. Views
  keep the source allocation alive until the result is released.
- Hex/UUID case conversion switches between compatible existing formats without
  touching payload bytes. Other compressed ASCII formats decode into the final
  result only when necessary. Raw and unknown formats keep Unicode fallbacks.
- LIKE handles CString and BString values and checks mandatory pattern literals
  against precomputed alphabet bitsets. Escapes, wildcards, case folding and
  padding retain their established semantics. Pattern-to-nibble compilation and
  caching of prepared query constants remain future work.
- FNV and structural hashing stream longer text with a local hash accumulator;
  small values retain the scalar decoder. MD5/SHA-1/SHA-256 stream from 256 text
  bytes onward, preserving the original short/plain path and its JIT emitter.
  Scheme serialization also processes compressed values in bounded chunks.
- Base64 encoding produces a lazy BString. Standard BString decoding returns
  its immutable raw bytes without encoding them first. URL-safe input still
  passes through the standard decoder so invalid alphabet characters fail.
  Hex operations consume compressed text directly.
- Interpreter and JIT accept both compressed representations as self-evaluating
  constants, including results of constant folding.

No disk layout, format ID, storage reader, locking rule or persistence cleanup
path changes in this follow-up. Existing legacy fixtures remain in the targeted
regression run. Regex, JSON, general replacement/splitting and language-specific
collations are outside this follow-up and may still materialize full strings.

Validation compares compressed and plain results (including error outcomes)
across legacy/new nibble formats, offsets, UUID case, both Base64 alphabets,
padding, empty values, Unicode patterns, chunk boundaries and GC lifetime.
Dedicated JIT tests exercise constants and composed operator calls; SQL tests
cover stored and dynamically produced Base64 values and invalid decoding.

### Follow-up A/B measurements

Measured on 2026-09-09 against `efb20afac` (current master after #844),
on an AMD Ryzen 9 7900X3D. Both processes used CPU affinity 0–3 and
`GOMAXPROCS=4`. Native operator and comparison benchmarks used Go 1.24.0,
the identical committed fixture, automatic `testing.B` calibration, and
200 ms per case in baseline/candidate/candidate/baseline order. Values below
are means of the two blocks per version. CString cases use ordered lower hex
with nibble offset 1. Size is 16 or 2048 text bytes for plain/CString, and
16 or 2048 raw bytes (24 or 2732 Base64 characters) for BString.

The LIKE microcase uses the impossible literal pattern `%~%`; substring requests
eight characters at offset one, concat appends `!`, and trim inputs need no
trimming.

SQL used the JIT Go toolchain `84fe25ee`, the same fixed 4096-row fixture,
5 warmup requests and 41 measured requests per query, also in ABBA order.
Reported times average the two block medians. The queries are the committed
composed projection and Base64 roundtrip performance cases, with only their
warmup/repetition counts changed for this manual run.

View-producing microbenchmarks measure the operator; subsequent decoding is
charged to its consumer. SQL measurements include result output. The bulk
projection path can already supply plain strings, so SQL figures describe
complete query chains rather than isolated CString decoding.

| SQL workload (4096 rows) | Baseline | Candidate | Change |
|---|---:|---:|---:|
| Compressed text composed projection | 7.25 ms | 7.15 ms | -1.4% |
| Compressed text Base64 roundtrip projection | 5.85 ms | 5.15 ms | -12.0% |

4096-value index sort: 1.4421 → 1.4448 ms (+0.2%). Identical CString/CString Less: 22.51 → 22.48 ns (-0.2%). None of the 78 operator or 18 comparison/sort cases exceeded a 20% regression in this run.

#### Operator measurements

| Case | Baseline ns/op | Candidate ns/op | Change | Baseline → candidate B/op |
|---|---:|---:|---:|---:|
| `plain/16/string?` | 19.93 | 2.67 | -86.6% | 16 → 0 |
| `plain/16/strlen` | 5.62 | 5.64 | +0.3% | 0 → 0 |
| `plain/16/concat` | 123.90 | 70.73 | -42.9% | 112 → 80 |
| `plain/16/substr` | 9.64 | 8.97 | -6.9% | 0 → 0 |
| `plain/16/toUpper` | 76.83 | 48.31 | -37.1% | 16 → 16 |
| `plain/16/strtrim` | 9.94 | 7.56 | -23.9% | 0 → 0 |
| `plain/16/base64_encode` | 109.00 | 6.50 | -94.0% | 48 → 0 |
| `plain/16/bin2hex` | 92.72 | 45.48 | -51.0% | 64 → 64 |
| `plain/16/strlike_cs` | 48.27 | 36.60 | -24.2% | 0 → 0 |
| `plain/16/fnv_hash` | 63.61 | 35.44 | -44.3% | 16 → 16 |
| `plain/16/stable_structural_hash` | 118.97 | 51.94 | -56.3% | 32 → 32 |
| `plain/16/sha256` | 239.20 | 129.00 | -46.1% | 128 → 128 |
| `plain/16/base64_decode` | 43.56 | 42.56 | -2.3% | 32 → 32 |
| `cstring/16/string?` | 40.94 | 2.48 | -93.9% | 32 → 0 |
| `cstring/16/strlen` | 2.13 | 2.04 | -4.3% | 0 → 0 |
| `cstring/16/concat` | 177.55 | 112.00 | -36.9% | 144 → 80 |
| `cstring/16/substr` | 25.87 | 26.39 | +2.0% | 8 → 0 |
| `cstring/16/toUpper` | 69.16 | 12.80 | -81.5% | 32 → 0 |
| `cstring/16/strtrim` | 29.93 | 31.27 | +4.4% | 16 → 0 |
| `cstring/16/base64_encode` | 65.70 | 27.55 | -58.1% | 64 → 16 |
| `cstring/16/bin2hex` | 68.82 | 69.84 | +1.5% | 80 → 32 |
| `cstring/16/strlike_cs` | 68.22 | 29.62 | -56.6% | 0 → 0 |
| `cstring/16/fnv_hash` | 56.56 | 58.28 | +3.0% | 32 → 32 |
| `cstring/16/stable_structural_hash` | 99.04 | 74.82 | -24.5% | 48 → 48 |
| `cstring/16/sha256` | 150.35 | 155.65 | +3.5% | 144 → 144 |
| `cstring/16/base64_decode` | 65.80 | 64.89 | -1.4% | 48 → 48 |
| `bstring/16/string?` | 66.00 | 2.73 | -95.9% | 64 → 0 |
| `bstring/16/strlen` | 44.49 | 3.15 | -92.9% | 48 → 0 |
| `bstring/16/concat` | 208.20 | 112.35 | -46.0% | 232 → 104 |
| `bstring/16/substr` | 48.33 | 44.73 | -7.4% | 48 → 8 |
| `bstring/16/toUpper` | 110.40 | 67.67 | -38.7% | 72 → 24 |
| `bstring/16/strtrim` | 45.50 | 19.49 | -57.1% | 48 → 0 |
| `bstring/16/base64_encode` | 87.17 | 44.15 | -49.3% | 112 → 48 |
| `bstring/16/bin2hex` | 95.62 | 77.56 | -18.9% | 144 → 48 |
| `bstring/16/strlike_cs` | 73.16 | 20.21 | -72.4% | 48 → 0 |
| `bstring/16/fnv_hash` | 83.09 | 85.88 | +3.4% | 64 → 64 |
| `bstring/16/stable_structural_hash` | 133.20 | 101.85 | -23.5% | 80 → 80 |
| `bstring/16/sha256` | 181.45 | 173.35 | -4.5% | 176 → 176 |
| `bstring/16/base64_decode` | 90.43 | 2.37 | -97.4% | 88 → 0 |
| `plain/2048/string?` | 19.59 | 2.81 | -85.6% | 16 → 0 |
| `plain/2048/strlen` | 5.53 | 5.32 | -3.8% | 0 → 0 |
| `plain/2048/concat` | 835.55 | 707.30 | -15.3% | 5184 → 5152 |
| `plain/2048/substr` | 8.72 | 9.38 | +7.6% | 0 → 0 |
| `plain/2048/toUpper` | 3464.50 | 3560.00 | +2.8% | 2048 → 2048 |
| `plain/2048/strtrim` | 6.92 | 7.70 | +11.3% | 0 → 0 |
| `plain/2048/base64_encode` | 1937.50 | 6.68 | -99.7% | 6144 → 0 |
| `plain/2048/bin2hex` | 2795.00 | 3267.50 | +16.9% | 8192 → 8192 |
| `plain/2048/strlike_cs` | 50.01 | 50.26 | +0.5% | 0 → 0 |
| `plain/2048/fnv_hash` | 1813.00 | 1808.50 | -0.2% | 16 → 16 |
| `plain/2048/stable_structural_hash` | 4794.00 | 1829.50 | -61.8% | 32 → 32 |
| `plain/2048/sha256` | 1277.50 | 1300.00 | +1.8% | 2176 → 2176 |
| `plain/2048/base64_decode` | 1436.50 | 1446.50 | +0.7% | 3072 → 3072 |
| `cstring/2048/string?` | 1028.75 | 2.69 | -99.7% | 2064 → 0 |
| `cstring/2048/strlen` | 2.12 | 2.41 | +13.6% | 0 → 0 |
| `cstring/2048/concat` | 2883.50 | 1648.00 | -42.8% | 9280 → 5280 |
| `cstring/2048/substr` | 25.46 | 25.77 | +1.2% | 8 → 0 |
| `cstring/2048/toUpper` | 4546.50 | 12.65 | -99.7% | 4096 → 0 |
| `cstring/2048/strtrim` | 995.50 | 30.67 | -96.9% | 2048 → 0 |
| `cstring/2048/base64_encode` | 2855.50 | 934.70 | -67.3% | 8192 → 2048 |
| `cstring/2048/bin2hex` | 3677.00 | 2931.00 | -20.3% | 10240 → 4096 |
| `cstring/2048/strlike_cs` | 878.75 | 30.59 | -96.5% | 0 → 0 |
| `cstring/2048/fnv_hash` | 2850.50 | 2592.00 | -9.1% | 2064 → 16 |
| `cstring/2048/stable_structural_hash` | 5831.50 | 2609.00 | -55.3% | 2080 → 32 |
| `cstring/2048/sha256` | 2279.50 | 2011.50 | -11.8% | 4224 → 1056 |
| `cstring/2048/base64_decode` | 2447.00 | 2375.00 | -2.9% | 5120 → 5120 |
| `bstring/2048/string?` | 1884.50 | 2.90 | -99.8% | 6160 → 0 |
| `bstring/2048/strlen` | 1835.50 | 3.19 | -99.8% | 6144 → 0 |
| `bstring/2048/concat` | 4235.50 | 2490.00 | -41.2% | 15424 → 9376 |
| `bstring/2048/substr` | 1881.00 | 45.12 | -97.6% | 6144 → 8 |
| `bstring/2048/toUpper` | 7425.50 | 3373.00 | -54.6% | 9216 → 3072 |
| `bstring/2048/strtrim` | 1856.50 | 20.04 | -98.9% | 6144 → 0 |
| `bstring/2048/base64_encode` | 4377.00 | 1736.50 | -60.3% | 14336 → 6144 |
| `bstring/2048/bin2hex` | 5498.50 | 3838.00 | -30.2% | 18432 → 6144 |
| `bstring/2048/strlike_cs` | 1916.00 | 20.66 | -98.9% | 6144 → 0 |
| `bstring/2048/fnv_hash` | 4190.50 | 3584.00 | -14.5% | 6160 → 16 |
| `bstring/2048/stable_structural_hash` | 8477.50 | 3622.50 | -57.3% | 6176 → 32 |
| `bstring/2048/sha256` | 3610.50 | 2661.00 | -26.3% | 9344 → 1056 |
| `bstring/2048/base64_decode` | 3959.00 | 2.25 | -99.9% | 10496 → 0 |

#### Comparison controls

| Case | Baseline ns/op | Candidate ns/op | Change | Baseline → candidate B/op |
|---|---:|---:|---:|---:|
| `Compare/difference-0/equal-CS` | 24.18 | 23.57 | -2.5% | 0 → 0 |
| `Compare/difference-0/equalSQL-CS` | 25.64 | 24.25 | -5.5% | 0 → 0 |
| `Compare/difference-0/less-CS` | 22.59 | 22.39 | -0.9% | 0 → 0 |
| `Compare/difference-0/less-CC` | 23.24 | 22.82 | -1.8% | 0 → 0 |
| `Compare/difference-31/equal-CS` | 35.31 | 34.96 | -1.0% | 0 → 0 |
| `Compare/difference-31/equalSQL-CS` | 43.08 | 41.84 | -2.9% | 0 → 0 |
| `Compare/difference-31/less-CS` | 34.82 | 33.59 | -3.5% | 0 → 0 |
| `Compare/difference-31/less-CC` | 23.39 | 22.71 | -2.9% | 0 → 0 |
| `Compare/difference-63/equal-CS` | 50.10 | 48.03 | -4.1% | 0 → 0 |
| `Compare/difference-63/equalSQL-CS` | 66.03 | 61.39 | -7.0% | 0 → 0 |
| `Compare/difference-63/less-CS` | 46.77 | 46.12 | -1.4% | 0 → 0 |
| `Compare/difference-63/less-CC` | 22.84 | 22.77 | -0.3% | 0 → 0 |
| `Compare/difference-64/equal-CS` | 49.05 | 48.70 | -0.7% | 0 → 0 |
| `Compare/difference-64/equalSQL-CS` | 64.06 | 64.74 | +1.1% | 0 → 0 |
| `Compare/difference-64/less-CS` | 46.70 | 47.91 | +2.6% | 0 → 0 |
| `Compare/difference-64/less-CC` | 22.51 | 22.48 | -0.2% | 0 → 0 |
| `Compare/uuid-less-CC` | 24.80 | 25.05 | +1.0% | 0 → 0 |
| `IndexSort` | 1442095.00 | 1444756.00 | +0.2% | 116 → 115 |
