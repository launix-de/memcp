<!-- Copyright (C) 2026 Carl-Philip Hänsch -->
# LZ4 dictionaries and reader hints

A read of a historically compressed StorageString dictionary expands the entire
LZ4 payload. Block dictionaries instead return `tagLZ4` (`scm.TagLZ4`, value 26)
handles: a GC-visible pointer to the immutable buffer and an auxiliary word with
a 32-bit logical byte offset and 24-bit length. The existing entry starts and
lengths remain the directory; there is no duplicate runtime offset array.
Compression borrows the existing raw dictionary during construction and drops
it afterward. Blocks are independently encoded in 64 KiB units.

## How the operator selects decoding

`GetCachedReader(hints)` resolves the representation during preparation:

| Hint | Behavior | Consumers |
|---|---|---|
| `ReadCompressed` | Return compressed handles even when a decoded cache exists. | Bigram walks and explicit compressed inputs. |
| `ReadAuto` | Reuse a ready decoded cache within an owned statement; otherwise return compressed handles. Never initiate full decoding or wait for optional cache reuse. | Residual scan filters and index probes. |
| `ReadDecoded` | Prepare the shared decoded dictionary once and lease it. | Sorted-index construction, ORDER BY, RecMap ordering and full rebuild replay. |

Prepared readers are the actual scalar and bulk scan inputs; direct storage
emitters must not bypass a decoded reader. Typed filters and map reducers use
that reader's physical tag. Physical delta rows still use their delta reader,
while computed deltas retain their transaction-bound reader.

Dense full-scan or range comparisons of short dictionaries (at most 256 logical
bytes per dictionary entry) request `ReadDecoded` for populations above the
1,024-row scan buffer. Point, IN/RecSet and bigram-pruned accesses keep
`ReadAuto`; early-exit probes do not initiate a complete decode.

Projection preparation starts with `ReadAuto`. Only an already-selected point
or consumed batch can request full decoding: short-key dictionaries up to
1 MiB may be decoded for a consumed point; larger ones need selectedRows ×
4 KiB to cover their logical size. These are bounded preparation heuristics,
not a claim that all compressed accesses are faster. Long report texts do not
meet the short-key condition. Preparation occurs before the selected batch's
row callbacks, or at the explicit already-selected point consumer. Reader
replacement refreshes compiled accessors and physical-tag specialization.
`ReadAuto` itself still never admits a new cache.

Full rebuild replay and physical shard reordering also prepare decoded readers
once, release their leases after the phase, and retain the compressed primary
payload. Reusing one reader across scan/build passes avoids repeated block work.

Identical immutable compressed handles compare equal without decoding. This
avoids repeated block work for shared dictionary keys in grouping.

These compressed-handle decisions apply to eligible raw text. Specialized inner
encodings currently need full cached decoding for access.

A bigram walk reuses one invocation-local scratch buffer and decodes blocks as
it walks them. It does not retain a complete decoded dictionary. The resulting
postings restrict subsequent scans; only candidate values used by residual
checks or projection are accessed. The implementation reads decoded bytes from
blocks, rather than comparing the LZ4 encoding itself.

Sorting expands the dictionary once before collecting comparison keys. The
published index retains its row permutation and stable compressed boundary
values, not views into the decoded arena. After cache eviction, probes can still
use that index and decode only the blocks they touch. An existing decoded cache
can accelerate probes without another full decode.

The complete decoded dictionary is an optional `TypeStringDict` cache child.
Active readers hold leases; eviction uses TryLock and declines cleanup while
leased. Statement leases are released by panic-safe `endQuery`; other consumers
must defer `ReleaseColumnReader`. Distinct decoded-generation identities prevent
late removal of an old cache from removing its replacement. Parent accounting
uses immutable payload sizes and excludes the separately charged cache.
Usage timestamps change during preparation, not comparison loops. Prefix-build
sampling is cleared after construction so it cannot retain a predecessor's raw
buffer.

All new non-empty dictionaries are block-compressed during private construction,
including Hex/nibble and Base64/binary inner encodings. There is no opt-in or
read-count gate. Construction pays block encoding once per generation; the
operator timings below exclude this cost. Cold loading normalizes historical raw dictionaries before
publication; historical monolithic LZ4 can remain compressed until its next save.
V0–V4 readers remain intact. The current V5 writer always writes a block buffer,
without a compression flag or redundant offsets, regardless of input version or
cache residency. Saving a published historical generation converts privately and
does not mutate its handles or admit a decoded cache. Empty dictionaries also
use the V5 container on disk. Incompressible blocks are literal inside that
container; LZ4 cannot guarantee a smaller payload for arbitrary bytes.

Eligible raw text returns compressed handles. Specialized inner encodings and
raw values exceeding the handle length limit use the common decoded cache for
access; their persisted and primary resident dictionaries are still LZ4 blocks.
The block container has a 32-bit logical-size limit. Historical raw dictionaries
exceeding that limit remain readable in their old resident representation; the
current writer rejects such a payload before writing, rather than truncating it.
The cache is never serialized. Existing file versions are read, not rewritten
merely by loading: migration reaches disk on the next normal save/checkpoint.

## Measured memory and CPU tradeoffs

Measurements dated 2026-10-13 use a private local report-text corpus covering a
complete partition interval: 16,663 non-null text values, 107,494,059 UTF-8 bytes.
No customer identity, document contents, database name or local path is
published. Hardware: AMD Ryzen 9 7900X3D. Toolchain: Go 1.26.0, linux/amd64,
`nophp`, ordinary Go execution. Two repetitions, 200 ms benchmark target; large
operations run fewer iterations. Values, complete bigram postings and sort order
are checked against the raw baseline before reporting results.

This isolates StorageString by storing the texts directly. It excludes
OverlayBlob/gzip behavior, HTTP, SQL planning, collation-key generation, process
RSS and production-database extrapolation. Timings are medians of the two
reported samples. A cold decoded cache means it is evicted each iteration;
compressed bytes and the allocator may remain warm. This is not process-cold or
power-on latency. `B/op` measures allocation volume, not peak resident memory.

| Storage measurement | Raw | LZ4 blocks |
|---|---:|---:|
| Resident dictionary and metadata, without decoded cache | 103.13 MB | 13.51 MB |
| Serialized storage | 103.13 MB | 13.51 MB |

The compact resident representation saves **86.9%**; V5 also saves **86.9%**
on disk against the historical raw V3 fixture. Both benchmark representations
now persist compressed when using the production writer, so the disk baseline
explicitly uses a test-only historical writer. A full decoded cache
adds approximately 103 MB while retained; the saving is not claimed while that
cache is resident. Memory pressure can reclaim the decoded child without losing
the compressed dictionary or its persisted data.

| Compressed-only access | Raw | LZ4 blocks |
|---|---:|---:|
| Full equality against plain text | 0.63 µs | 11.51 µs |
| Adjacent lexical comparison | 0.04 µs | 15.68 µs |
| Random LIKE predicate | 19.90 µs | 66.86 µs |

These comparisons allocate zero heap objects. LIKE allocates zero objects for
the compressed path; the raw baseline allocates lowercase text. Direct block
comparisons cost more CPU, especially for short lexical prefixes. This is why
sorting requests `ReadDecoded` instead of repeatedly decoding random blocks.

| Whole-corpus operation | Raw | LZ4 blocks |
|---|---:|---:|
| Bigram build | 498.3 ms | 1059.1 ms |
| Allocation volume during bigram build | 122.35 MB | 14.23 MB |
| Sort using compressed handles throughout | 10.4 ms | 3249.2 ms |
| Sort including one complete decode, cache evicted each iteration | 10.7 ms | 52.2 ms |
| Sort with decoded cache already present | 10.7 ms | 10.1 ms |

Bigram building trades approximately 2.13× CPU time for substantially less
allocation and a compact retained dictionary. Choosing full decoding for sorting
reduces LZ4 sort time from 3.25 s to
52.2 ms including decoding, or
10.1 ms with a ready cache. The benefit depends on
operator demand; the compressed-only path is not a universal CPU speedup.

Read-only comparisons, LIKE, hashing, substring/trim views and cursor traversal
use the tag directly. Operations producing new text allocate their output.
Regex and numeric-conversion fallbacks still materialize their input; this
change does not claim allocation-free behavior for those fallbacks.

## Evidence and reproduction

[All numeric samples](lz4-dictionary-performance.csv) and
[storage sizes and samples](lz4-dictionary-performance.json) contain only
aggregated results. The private corpus is intentionally excluded from Git.
To use an independently supplied JSON array of strings:

```sh
MEMCP_STRING_BENCH_CORPUS=/path/to/private-texts.json \
  go test -tags=nophp ./storage -run '^$' \
  -bench '^BenchmarkStorageStringBufferCorpus$' -benchtime=200ms -count=2 -v
```

Local validation uses focused operator, persistence, cached JIT reader,
prepared-filter tag, Unicode/block-boundary, cache-lease, eviction-ledger and
concurrent-accounting tests, including the race detector. Full SQL, upgrade,
JIT and performance suites run in CI.
