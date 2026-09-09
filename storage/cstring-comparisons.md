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
