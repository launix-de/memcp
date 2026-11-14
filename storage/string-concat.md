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
| Build | 12.499 ms | 10.238 ms | 14.505 ms |
| Random binary comparison | 4695.000 ns | 14.190 ns | 68.920 ns |
| Append to reused output | 2387.000 ns | 2.732 ns | 79.890 ns |
| Sort 4,096 handles | 236.557 ms | 0.668 ms | 3.486 ms |
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
| read-1 | 9.654, 7.814, 7.516 | 8.881, 7.375, 7.524 | 8.269, 4.86, 4.837 |
| read-0 | 9.975, 7.601, 7.573 | 8.327, 7.469, 7.452 | 4.873, 4.871, 5.013 |
| read-2 | 6.447, 6.372, 6.602 | 8.083, 7.372, 7.392 | 6.892, 4.645, 4.727 |
| build | 1.24994e+07, 1.20187e+07, 1.26757e+07 | 1.02379e+07, 1.09493e+07, 1.02102e+07 | 1.4496e+07, 2.61117e+07, 1.45051e+07 |
| compare | 4695, 4723, 4662 | 17.66, 10.35, 14.19 | 68.92, 66.84, 80.98 |
| append | 2387, 2375, 2439 | 2.732, 2.704, 2.733 | 74.73, 79.89, 81.82 |
| sort | 2.39935e+08, 2.36557e+08, 2.35548e+08 | 654976, 667753, 679946 | 3.51379e+06, 3.42115e+06, 3.48629e+06 |

Build allocation samples (`bytes/op`, `allocs/op`):

- lz4: (13276896, 96), (13273253, 92), (13273221, 92)
- raw: (12802309, 69), (12804168, 71), (12804168, 71)
- concat: (14595802, 7745), (14595840, 7745), (14593864, 7742)
