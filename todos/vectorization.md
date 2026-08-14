# Copyright (C) 2026  Carl-Philip Haensch

# JIT, SIMD Bulk Operations, and Hybrid Scan Execution

## Status: the backend decision is deliberately open

This document does not select a vector interpreter or a query-specific JIT as
the universally best execution model. That decision has not been measured for
MemCP yet.

The design space contains three physical execution candidates:

1. **Fine-grained JIT:** the approach in `todos/jit.md`. Generate one
   shard-specialized native loop from column decoders and the small Scheme
   operations in filter, map, and reduce. Values remain scalar machine values;
   there is no vector-operator dispatch and no intermediate selection vector
   unless the expression needs one.
2. **Precompiled SIMD bulk operators:** a MonetDB/DuckDB-style interpreter of
   typed column operations. The executor dispatches to already compiled AVX2,
   AVX-512, or scalar kernels. Operators exchange a shard-relative selection
   bitmap or sparse record-ID vector. Startup is cheap, but every operator
   boundary may require another pass or a materialized mask.
3. **SIMD JIT:** generate a query- and storage-specific fused SIMD loop. This
   can remove bulk-dispatch and intermediate-mask costs, but needs a materially
   more expensive compiler, vector register allocation, and more code-cache
   space than the fine-grained scalar JIT.

All three may be useful. OLTP point/range queries, short ERP reports, and large
TPC-H scans have very different break-even points. The physical planner should
eventually compare measured candidates; this document first defines how to
build and measure them without baking the answer into planner semantics.

The previous version of this document incorrectly selected precompiled kernels
first and relegated JIT to a later optimization. There was no MemCP-specific
measurement supporting that choice.

## Relation to `todos/jit.md`

`todos/jit.md` remains the source of truth for the fine-grained JIT:

- a main-storage loop specialized per shard;
- a separate, normally interpreted delta loop;
- direct decoding through storage-specific emitters;
- `Declaration.JITEmit` for small Scheme operations;
- `JITValueDesc` propagation instead of `Scmer` in the hot loop;
- storage fields such as data pointer, bit width, offset, NULL code, and row
  count embedded as immediates;
- shard-local executable pages and bounded code lifetime.

The vectorization work is not a replacement for that design. It adds two
questions that `jit.md` leaves open:

- Can precompiled SIMD bulk kernels amortize their dispatch and mask traffic
  well enough to beat the fine-grained JIT, especially when JIT compilation is
  included?
- If not, or only for some shapes, does a SIMD-emitting extension of that JIT
  pay for its larger compiler?

The existing repository currently implements only part of `jit.md`:
`JITValueDesc`, the amd64 writer/compiler scaffolding, and a small number of
`Declaration.JITEmit` registrations exist. The storage-specific scan-loop JIT,
generated `Storage.JIT` methods, shard JIT pools, and SIMD emitters described in
the TODO are not complete. Benchmarks must therefore compare implemented
vertical slices, not extrapolate from the TODO's cycle estimates.

## Orthogonal choices: entry point and backend

The two entry variants from the original request do not imply a backend:

### A. Physical planner lowering

`build_queryplan` knows SQL-level physical facts: estimated cardinality and
selectivity, required ordering, LIMIT/braking, join shape, expected reuse, and
whether a scan feeds an aggregate or materialized result. It can produce a
typed scan specification and a scalar fallback.

It must not choose AVX instructions or assume one concrete storage encoding.
Those differ per shard and may change after rebuild. The physical operator may
carry a ranked set of backend families and defer the final per-shard dispatch
until the concrete storages are locked and known.

### B. `scan` / `scan_order` optimizer-hook recognition

The storage optimizer hooks can recognize an existing filter/map/reduce
callback pipeline and derive the same typed scan specification. This is needed
for non-SQL Scheme callers and for physical plans that still lower to ordinary
`scan` calls.

The hook may prove purity, types, NULL behavior, reducer ownership, and
associativity. It must not redo join ordering, predicate ownership, outer-join
barriers, ordering, or LIMIT decisions.

### Backends after A or B

Both entry points end at a semantic `ScanSpec`, not at a mandatory vector IR:

```text
SQL physical lowering (A) ----+
                              +--> typed ScanSpec + scalar fallback
scan hook recognition (B) ----+              |
                                             +--> fine-grained JIT
                                             +--> SIMD bulk program
                                             +--> SIMD JIT
                                             +--> existing interpreter
```

The shared contract should include normalized expressions, physical/logical
types, decimal scale, NULL semantics, referenced columns, candidate source,
ordering, reducer semantics, and a stable fingerprint. Backend-specific IRs
are allowed and expected:

- the fine-grained JIT can continue compiling expressions directly through
  `JITValueDesc` and `Declaration.JITEmit`;
- the bulk backend needs a short program of typed kernel calls and explicit
  selection carriers;
- the SIMD JIT needs vector descriptors, masks, and vector registers.

Forcing all three through an interpreted vector bytecode would add overhead to
the JIT path without evidence. Duplicating SQL semantics in three compilers
would be equally wrong. The common layer ends at proven semantics and typed
operations; machine-level lowering remains backend-owned.

## Planner boundary

The existing phase boundary remains:

```text
parser AST
  -> untangle_query
  -> join_reorder / logical optimize
  -> build_queryplan
  -> physical ScanSpec
  -> per-shard backend selection and execution
```

Logical planning may carry cardinality, selectivity, confidence, predicate
dependencies, NULL-extension barriers, required order, LIMIT, and reuse. These
are logical or abstract physical requirements and are relevant to reorder.

It must not contain `scan`, bulk kernel names, SIMD masks, `StorageInt` bit
widths, RecSets, ORC columns, or JIT entry points. A/B and JIT/bulk selection
start only in physical lowering or generic operator optimization.

## Exact MemCP scan integration

Today `storageShard.scan` obtains condition column readers, iterates candidate
record IDs through `iterateIndex`, checks visibility/deletions, calls
`ColumnStorage.GetValue` per referenced row and column, invokes the scalar
condition, and feeds passing IDs to `mapper.Stream`. The default index transport
buffer is 1,024 IDs; that buffer is not a useful vectorization domain.

The new path should sit immediately after candidate boundaries and referenced
storages have been resolved, before the per-record `GetValue` loop:

```go
type ScanSpec struct {
	// normalized typed filter/map/reduce descriptions
	// reducer merge and ownership contract
	// order, limit and candidate-source properties
	// original optimized callbacks as exact fallback
}

func (t *storageShard) runSpecializedMainScan(
	spec *ScanSpec,
	boundaries scanBoundaries,
	storages []ColumnStorage,
) (partial scm.Scmer, matched uint, ok bool)
```

Names and exact signatures are not fixed, but the control flow is:

1. acquire shard read rights and load storages through existing helpers;
2. take the required shard read lock and snapshot visibility/candidate state;
3. classify the main-storage candidate domain as dense range, monotonic sparse
   IDs, irregular ordered IDs, or unsupported;
4. inspect the concrete storage types and their encoding parameters;
5. select and run interpreter, fine JIT, bulk SIMD, or SIMD JIT;
6. evaluate the small delta through the current scalar path unless measurement
   justifies a delta backend;
7. merge main and delta partials with the existing `reduce2` and ownership
   contract.

There must be no function call, interface dispatch, allocation, lock operation,
or `Scmer` construction per main-storage row on a supported specialized path.

### Lifetime and rebuild safety

`jit.md` proposes embedding storage and deletion-bitmap pointers as immediates.
That is only safe while those exact objects remain alive and immutable. A shard
rebuild, lazy load/eviction, or storage replacement can invalidate such code.

Therefore either:

- compile and execute under one acquired read capability with strong references
  to all storages, and discard the code before releasing it; or
- cache code only with a shard storage-generation key, retain the referenced
  storages, and reject execution when the generation differs.

A persistent query-plan cache must never own raw storage pointers. Bulk kernels
have the same lifetime rule for borrowed slices, even though their code itself
is reusable.

## The shard is the relational vector, not an allocated value matrix

`Settings.ShardSize` is currently 60,000. One shard is the scheduling,
visibility, and selection domain. It does not mean allocating decoded arrays of
60,000 values for each expression.

- SIMD instructions still consume register-width lane groups.
- A dense selection is one shard-sized bitmap (about 7.5 KiB at 60,000 rows).
- A sparse selection is a monotonic `[]uint32` candidate stream when already
  provided by an index or RecSet.
- Fine JIT can avoid a selection entirely for a fused filter/aggregate.
- Bulk execution may materialize a bitmap between kernels; its cost must be
  counted.
- SIMD JIT may keep masks in vector registers inside a fused loop and only
  materialize them when another operator consumes the selection.

Full and contiguous range scans should iterate `[lo, hi)` directly rather than
round-trip through the 1,024-record index buffer. Unique probes and short
ordered LIMIT paths normally remain scalar. Sparse gather is a measured choice,
not a default: cache-cold AVX2 gathers can lose to a scalar monotonic loop.

## Concrete column-storage lowering

The backend selector must specialize on MemCP's actual `ColumnStorage` formats.
`ColumnStorage.GetValue` remains the compatibility fallback, not the fast-path
API. A new generic `VectorColumn` interface returning decoded spans would hide
the format information needed by the JIT and could force materialization. Use
a type switch once per column and shard, then call format-specific compiler or
kernel builders.

### `StorageInt`

`StorageInt` stores packed unsigned codes in `chunk []uint64`, with `bitsize`,
`offset`, optional `hasNull`, and a raw `null` code. `GetValueUInt` computes
`bitpos = row * bitsize`, combines two words when the value crosses a word
boundary, and right-aligns the result.

- Fine JIT embeds chunk pointer, bit width, offset, NULL code, and count and
  emits the exact packed decoder into the row loop as described in `jit.md`.
- Bulk SIMD needs precompiled unpack/compare/reduce kernels. Widths
  1/2/4/8/16/32/64 deserve direct kernels; arbitrary widths need either a
  generated/unrolled decoder, a kernel table, or a scalar packed decoder.
- SIMD JIT can generate the arbitrary-width extraction with immediate shifts
  and fuse comparisons before decoded values leave registers.
- Predicate constants should be converted to raw code space once. For SUM,
  accumulate raw non-NULL values and add `count * offset`; do not create decoded
  `int64` arrays.

Odd bit widths are an important comparison case: the smaller memory footprint
may beat easy SIMD on wider decoded values because cache lines dominate.

### `StorageDecimal`

`StorageDecimal` wraps an embedded `StorageInt` plus `scaleExp`. Keep scaled
integers and scale metadata through filter and aggregate. Q6-style
`extendedprice * discount` needs a widened exact integer product and combined
scale; converting every value to float is not an acceptable shortcut. Overflow
or an unsupported scale combination must reject the fast path before output.

### `StorageFloat`

`StorageFloat` is contiguous `[]float64`; NaN represents SQL NULL. It is the
cleanest bulk-SIMD baseline: direct loads, unordered comparisons for validity,
and masked aggregates. It is also the case most favorable to SIMD JIT fusion,
so results from it must not be generalized to packed integer columns.

### `StorageConst`

Fold the predicate/projection once per shard. COUNT/SUM can use the selected row
count and one constant. No backend should loop over a constant column merely to
present a uniform API.

### `StorageSeq`

`StorageSeq` represents runs through packed record ID, start, and stride
columns. Per-row `GetValue` searches for the owning run. A dense scan should
walk runs once. Filters can solve ranges within a run, and SUM can use the
arithmetic-series formula. A monotonic sparse candidate stream should advance
one run cursor rather than binary-search for every record ID.

### `StorageEnum`

`StorageEnum` stores up to eight values and an rANS-coded symbol stream with L1
and L2 jumps. Comparisons, IN, grouping, and hashing should operate on symbol
IDs and translate to `Scmer` values only at the result edge. A single rANS state
has a serial dependency; SIMD requires decoding several independent chunks or
states in parallel. That constraint must be benchmarked rather than described
as an ordinary packed integer load.

### `StorageString`

`StorageString` uses packed dictionary IDs or packed start/length columns over
a byte dictionary/buffer, with raw, nibble, UUID, and Base64 encodings. Its
dictionary may be LZ4-compressed and lazily expanded.

- Resolve equality/IN constants to dictionary IDs once and compare packed IDs.
- Use dictionary IDs for shard-local grouping; cross-shard grouping needs a
  proven common dictionary or translation during merge.
- For non-dictionary strings, filter length first and use AVX byte comparison,
  prefix, or LIKE kernels directly on the encoded representation where valid.
- UUID/nibble/Base64 comparisons should avoid expanding to Go strings.
- Call `ensureDict` before entering native code when expansion is required;
  never take its lock or decompress inside the hot loop.
- Construct `CString`/`BString` only for selected output rows.

### `StoragePrefix`

`StoragePrefix` combines a packed prefix ID with suffix `StorageString` data.
Equality and prefix predicates should resolve prefix and suffix constraints
without concatenating a string per row. Unsupported lexical operations fall
back before the scan begins.

### `StorageSparse`

`StorageSparse` has packed record IDs and dynamic `[]scm.Scmer` values. Dense
execution should merge the candidate stream with one monotonic sparse-record
cursor; the complement provides the NULL mask. `IS NULL`, `IS NOT NULL`, and
selection can be bulk operations. Arithmetic on the dynamic values remains
scalar until the value payload itself has a typed columnar representation.

### `StorageSCMER`

`StorageSCMER` is the dynamic/uncompressed representation and compression
source. It is not a SIMD numeric source. A fine JIT may specialize stable tags,
but otherwise it uses the scalar fallback. Large persistent main columns should
normally have been converted by `proposeCompression` to const, sparse, enum,
string, sequence, integer, decimal, or float storage.

### `StorageComputeProxy`

If all requested rows are valid and the proxy is compressed, delegate to its
underlying typed storage. Otherwise split valid and invalid rows before native
execution. Invalid values may be computed by the current scalar callback and
then consumed; no native loop should jump into lazy recomputation per row.
Ordered computed columns must finish their required range preparation before a
borrowed fast path begins.

### `OverlayBlob`

Hash references may trigger external blob lookup and decompression. A native
path can use the base/hash only for operations whose semantics are valid on
that representation. Value comparison or output is a residual scalar fetch.
Blob I/O must never occur inside an AVX or generated scan loop.

### Main storage and delta

All three specialized candidates target compressed immutable main storage.
Delta rows remain dynamic and small; the default is the current scalar
interpreter followed by `reduce2`. This preserves transaction visibility and
avoids compiling for a handful of rows. Delta vectorization becomes a separate
proposal only if measurements show it matters.

## Three backend implementations

### Fine-grained scalar JIT

Implement the scan-loop portion already designed in `jit.md`. The generated
loop decodes one row's referenced columns, evaluates the filter, map, and
reducer with typed machine values, and advances. It is fused and has no
operator-level materialization. Its decisive questions are actual compile time,
instruction-cache footprint, branch behavior, and whether arbitrary packed
decoders can be emitted cheaply enough per shard.

The implementation should extend the existing files rather than create a
second JIT framework:

- storage-specific emission beside the corresponding storage type;
- scalar expression emission through `scm.JITContext`, `JITValueDesc`, and
  `Declaration.JITEmit`;
- scan-loop construction in storage with code lifetime tied to shard rights.

### Precompiled SIMD bulk interpreter

This backend compiles no machine code per query. A typed bulk program might be:

```text
int.compare_range(shipdate, raw_lo, raw_hi) -> mask0
decimal.compare_range(discount, raw_lo, raw_hi, mask0) -> mask0
decimal.compare_lt(quantity, raw_limit, mask0) -> mask0
decimal.mul_sum(extendedprice, discount, mask0) -> partial
```

The program is interpreted once per bulk operator, not once per row. Kernel
dispatch happens outside lane loops. Inputs remain encoded wherever the kernel
supports that encoding. The main costs are dispatch, repeated column passes,
selection bitmap writes/reads, and loss of fusion. The advantages are almost
zero query compile cost, small reusable code, mature hand-tuned kernels, and
predictable code-cache behavior under many short distinct queries.

Kernel tables should dispatch once on CPU level, storage encoding, bit width,
NULL mode, predicate, and selection representation. Scalar reference kernels
are mandatory; amd64 AVX2 is the first optimized target, AVX-512 only when
measured. ARM64 NEON can use the same semantic kernel program later.

### SIMD JIT

This backend extends the `jit.md` emitter with vector registers and mask/value
descriptors. It generates a fused loop which loads/decodes several rows,
evaluates all supported predicates, and updates the aggregate without writing
intermediate masks.

It should reuse semantic operation registration, storage-specialized constants,
W^X pages, fixups, safe points, and fallback rules from the scalar JIT. It needs
its own vector register allocator and amd64 VEX/EVEX emitters; pretending scalar
`JITValueDesc` is already a vector IR will make NULL masks and register pressure
implicit and unsafe.

SIMD JIT is not automatically the final stage. If compile time or code-cache
pressure loses on ERP query diversity, the bulk backend may remain preferable
even for large scans. Conversely, fine-grained scalar JIT may beat both on
packed odd-width data or short ranges.

## First decisive experiment: one vertical Q6 pipeline

Before designing joins or general vector bytecode, implement the same supported
Q6-shaped pipeline in all feasible backends:

```sql
SUM(extendedprice * discount)
WHERE shipdate >= ? AND shipdate < ?
  AND discount BETWEEN ? AND ?
  AND quantity < ?
```

Use the real MemCP encodings selected for these TPC-H columns, especially
`StorageInt`/`StorageDecimal`; do not benchmark only decoded `[]int64` or
`[]float64` arrays.

The candidates are:

- current callback interpreter;
- fine-grained scalar JIT, one fused row loop;
- precompiled SIMD bulk kernels, including actual mask traffic;
- SIMD JIT, only after the first two provide a justified target.

Measure separately:

- ScanSpec recognition/lowering;
- per-shard compile time and bytes of generated code;
- cold execution, warm execution, and repeated execution;
- cycles, instructions, branches/misses, LLC misses, and allocations;
- time per input row and per selected row;
- selection bitmap/scratch bytes;
- total query latency including main/delta merge and result conversion.

Run at row counts 0, 1, 8, 32, 128, 1K, 10K, and a full 60K shard, with
selectivities near 0%, 1%, 10%, 50%, and 100%. Repeat for contiguous range and
sparse candidate IDs. Include at least packed widths 3/5/7, 8/16/32/64,
nullable/non-nullable, and a delta tail.

The experiment must answer crossovers, not only throughput at one scale:

```text
T_interpreter = startup_i + N * row_i
T_fine_jit    = compile_j(expr, encoding) + N * fused_scalar_j
T_bulk        = startup_b + sum(kernel_dispatch + rows_touched * kernel_cost
                                + masks_written/read)
T_simd_jit    = compile_v(expr, encoding) + N * fused_simd_v
```

Fit and publish the measured constants with confidence ranges. Do not import
the speculative 25-row break-even from `jit.md` as a planner threshold.

## Planner cost and adaptive choice

The physical cost domain needs independent dimensions for:

```text
compile_ns
code_bytes
startup_ns
decode_ns_per_row by storage encoding
operation_ns_per_row
bulk_dispatch_ns
selection_write/read_ns_per_word
sparse_probe_ns_per_id
result_box_ns_per_selected_row
scratch_bytes
estimated input/output rows and confidence
expected executions/reuse
```

The planner can rank backend families using estimates, but concrete per-shard
encodings are runtime facts. A practical split is:

1. `build_queryplan` decides whether a scan is eligible for specialization and
   records estimated rows, reuse, order/LIMIT constraints, and allowed backend
   families;
2. after storage acquisition, the scan executor obtains concrete calibrated
   costs for that encoding and selects a backend per shard;
3. repeated executions feed telemetry into the existing cost facts without
   changing SQL semantics;
4. a cached compiled candidate is reused only when its expression fingerprint,
   CPU level, storage descriptors/generation, and semantic version match.

Expected behavior, subject to measurement:

- unique probes and early ordered LIMIT: interpreter or fine JIT only if cached;
- short ranges and high query diversity: bulk or interpreter may avoid compile
  cost;
- long scans with simple operations: precompiled SIMD bulk may win;
- long fused predicates/aggregates with mask traffic: SIMD JIT may win;
- odd-width packed integers: fine scalar JIT or encoding-specific SIMD,
  depending on cache and decode measurements;
- repeated prepared reports: amortized JIT compile cost may change the choice.

`EXPLAIN` should report candidate backend, estimated compile and execution cost,
storage assumptions, reuse assumption, selected fallback, and measured reason
when runtime dispatch changes the backend.

## `scan_order`, LIMIT, joins, and grouping

Ordered index traversal remains the owner of row order. Vectorization must not
turn it into an unordered dense scan. OFFSET/LIMIT and partition limits remain
hard boundaries; early braking normally beats bulk startup when only a few rows
are needed. Dense monotonic ranges can use any backend, while irregular ordered
IDs use sparse kernels or scalar execution and preserve input order.

The currently disabled `scan_order` batch rewrite must not be enabled until
final partial-batch, `reduce2`, outer-row, ordering, and LIMIT semantics have an
explicit contract.

Join probing, grouping, top-k, and typed sinks are later candidates. They reuse
the same backend comparison rather than assuming vector interpretation:

- a bulk backend may batch hash/index probes and group updates;
- a fine JIT may fuse key extraction with a specialized probe helper;
- SIMD JIT may inline hashing/probing where code size permits;
- dictionary IDs can stay encoded only with proven dictionary identity or
  translation;
- the logical join tree, predicate provenance, and outer barriers are never
  changed by backend selection.

## Correctness and fallback

Every specialized candidate retains the original optimized scalar callbacks.
Unsupported type, encoding, CPU feature, ownership proof, or storage generation
rejects the candidate before externally visible output. Mid-shard fallback may
only restart from a clean accumulator; it must not double-apply a partial
reducer.

Differential tests compare interpreter, fine JIT, bulk, and SIMD JIT where
implemented:

- values, order, multiplicity, and SQL NULLs;
- decimal scales, exact products, overflow, NaN, and signed zero;
- empty input, neutral ownership, outer rows, and `reduce2` merge;
- main storage plus inserts/deletions under every transaction mode;
- dense full/range, sparse RecSet/index, unique point, and ordered LIMIT;
- all concrete storage formats and their fallback paths;
- row counts around SIMD width, packed-word crossings, and shard boundaries;
- cold/cached code and storage rebuild between executions.

TPC-H correctness is necessary but not sufficient. Existing SQL suites in
`tests/` remain authoritative, and performance assertions should detect changes
in asymptotic behavior rather than depend on a narrow absolute timing window.

## Work packages and decision gates

### 1. Shared semantic ScanSpec and baseline

- Normalize typed filter/map/reduce semantics from A and B without a backend IR.
- Preserve original callbacks and reducer ownership/merge rules.
- Benchmark the current interpreter on Q6 and representative ERP point, range,
  ordered-LIMIT, and report shapes.

Acceptance: A and B produce the same fingerprint for equivalent scans; no
physical artifact leaks into logical planning; fixed/row/selection costs are
reported separately.

### 2. Fine-grained JIT vertical slice

- Complete only the `StorageInt`/`StorageDecimal` scan-loop pieces needed by Q6.
- Reuse the current JIT descriptors and small-operation emitters.
- Bind generated code lifetime to stable shard storages.

Acceptance: exact Q6 result, no per-row allocation or `Scmer`, compile/code-size
measurements, and clean scalar fallback for main/delta edge cases.

### 3. Precompiled SIMD bulk vertical slice

- Implement scalar reference and AVX2 packed numeric compare/masked-multiply-sum
  kernels for the same Q6 encodings.
- Account for every dispatch and selection bitmap pass.

Acceptance: exact differential results and crossover curves against interpreter
and fine JIT, including cold one-shot and repeated queries.

### 4. Backend decision and physical cost integration

- Decide from measurements whether both vertical slices remain production
  candidates.
- Add calibrated backend facts and per-shard encoding dispatch.
- Expose estimates and runtime choice in `EXPLAIN`/telemetry.

Acceptance: the choice predicts the measured winner within a defined tolerance
on held-out row counts, selectivities, encodings, and reuse counts; point/range
OLTP latency does not regress.

### 5. SIMD JIT decision gate

Implement vector emission only if bulk mask/dispatch overhead is material and
the estimated additional compiler/code-cache cost has a realistic crossover.
Prototype the same Q6 loop before generalizing vector descriptors.

Acceptance: total latency, including compilation, beats the selected scalar JIT
or bulk backend in a documented region large enough to justify maintenance.

### 6. Broader storage and operator coverage

Add formats and operations in measured benefit order: float, const/sequence,
enum/dictionary strings, ordered ranges, grouping, joins, and sinks. Sparse,
compute-proxy, dynamic SCMER, and blobs retain explicit residual paths.

Acceptance: each added path has encoding-specific differential tests, a
reproducible benchmark, zero allocations per supported main-storage row, and no
regression in compile time, code size, OLTP point/range latency, or transactional
correctness.

## Decision criteria

No backend is selected because it is fashionable or because one architecture
document mentioned it last. A backend remains only when it owns a measured
region of the workload space or provides a necessary portability/correctness
fallback. If both JIT and bulk survive, the query planner and per-shard executor
choose between them using the same measured physical cost domain.
