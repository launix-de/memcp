# Copyright (C) 2026  Carl-Philip Haensch

# Shard-wide Vectorized Query Execution

## Status and decision

This document proposes vectorized, SIMD-accelerated execution for physical
query plans. It does not change the logical planner model.

The recommended design combines both proposed entry points:

- **A: physical lowering** may deliberately select a vectorized scan pipeline.
- **B: the `scan` / `scan_order` optimizer hooks** may recognize a safe scalar
  callback pipeline and specialize it to the same vector backend.

There must be only one vector IR, one capability checker, and one executor.
Variant A is the authoritative cost-based SQL path. Variant B is an
opportunistic specialization and a fallback for plans produced outside the SQL
planner. It must not become a second physical planner.

The first implementation should use precompiled scalar/AVX kernels. Query JIT
may fuse the same vector IR later, but vectorization must not depend on JIT.

## Goals

- Eliminate per-row `Scmer` construction and callback dispatch from supported
  scan, filter, projection, and aggregate pipelines.
- Execute directly on column storage and its compressed representations where
  practical.
- Keep intermediate relational results in storage operators, RecSets, indexes,
  shard masks, or final result buffers; never materialize whole relations as
  Scheme lists.
- Preserve SQL NULL, decimal, overflow, visibility, ordering, outer-join, and
  transaction semantics exactly.
- Use one shard as the logical scheduling and vectorization unit.
- Provide a scalar implementation of every vector operation as the correctness
  oracle and fallback on non-amd64 platforms or unsupported storage types.
- Make physical choice and measured benefit visible in `EXPLAIN` and telemetry.

## Non-goals

- No vector artifacts in `untangle_query` or logical join reorder.
- No fixed 2,048-row execution chunks and no new chunk-based relational layer.
- No mandatory full-column decompression before execution.
- No attempt to vectorize arbitrary Scheme, side effects, mutation callbacks,
  correlated scans, or user functions in the first version.
- No AVX-only correctness path.

## Terminology

Three different sizes must not be conflated:

- **Shard vector:** the logical row domain of one storage shard, normally up to
  `Settings.ShardSize` rows (currently 60,000). Masks and vector expressions use
  shard-relative row IDs.
- **Storage span:** a contiguous or encoded region exposed by one column
  storage. A shard vector may consist of several spans without changing its
  identity.
- **SIMD lane group:** the 128/256/512-bit register-sized portion processed by
  one kernel iteration. This is an implementation detail, not a materialized
  execution batch.

A shard-wide vector therefore does not mean allocating 60,000 decoded values
for every expression. It means evaluating one pipeline over the shard while
keeping only borrowed column views, a selection representation, small reusable
kernel scratch space where unavoidable, and shard-local aggregate state.

## Phase boundary

The existing planner pipeline remains:

```text
parser AST
  -> untangle_query
  -> join_reorder / optimize
  -> build_queryplan
  -> scalar or vector physical execution
```

Logical planning may carry properties needed for the decision:

- estimated input and output cardinality
- selectivity and confidence
- required and available ordering
- LIMIT and braking opportunities
- expression types and NULLability
- expected reuse
- dense scan, range scan, sparse RecSet, or point-probe character

It must not contain `vec_scan`, vector masks, AVX instructions, storage spans,
or kernel names. These are physical artifacts chosen in `build_queryplan` or by
the generic operator optimizer.

## One backend, two entry points

### A. Cost-based physical lowering

`build_queryplan` may emit a vector-capable physical scan when all semantics are
known and the common cost domain predicts a benefit. Conceptually:

```scheme
(scan_vec tx table
	filter-columns filter-lambda
	map-columns map-lambda
	reduce-lambda neutral reduce2-lambda
	outer? scalar-fallback)
```

The exact syntax is deliberately not fixed here. It may instead be an optional
physical program argument to `scan`. The important contract is:

- the ordinary scalar plan remains available as an exact fallback;
- the vector program is produced only during physical lowering;
- plan selection considers compile cost, density, storage encoding, startup,
  output materialization, and estimated rows;
- join order, predicate ownership, outer barriers, ordering, and LIMIT remain
  unchanged.

This path can make choices that a local hook cannot infer reliably, such as
whether a sparse RecSet probe is cheaper than scanning a shard-wide mask, or
whether an ordered LIMIT should remain an index-driven scalar probe.

### B. `scan` and `scan_order` optimizer specialization

The existing optimizer hooks already inspect filter, map, reduce, and neutral
callbacks. They should additionally call the shared vector compiler after
ordinary callback optimization and type propagation.

The hook may specialize only when it can prove:

- every referenced operation has vector semantics registered in its
  declaration;
- callbacks are pure and deterministic for the duration of the scan;
- captured values are immutable scalar parameters or supported vector inputs;
- reducer associativity and shard merge semantics are declared and valid;
- NULL and overflow behavior has an exact vector implementation;
- no update/increment pseudo-column or other side effect is present;
- fallback preserves the original optimized scalar callbacks.

This path is useful for hand-written Scheme, RDF plans, computed-column
preparation, and older SQL lowering that still emits an ordinary scan. It must
not choose another join order, materialize a new relation, or override an
ordering decision made by the physical planner.

### Combined rule

Both paths call a single operation:

```text
CompileVectorPipeline(typed callbacks, scan properties)
  -> unsupported(reason)
  -> VecProgram + scalar fallback + capability requirements
```

An explicitly emitted vector candidate and an automatically recognized scalar
scan must produce the same `VecProgram` fingerprint. Compilation is cached by
callback structure, concrete input types, storage capabilities, CPU feature
level, and SQL-semantics version.

## Typed vector IR

The vector IR should be small, typed, and independent of AVX instruction names.
It describes a fused pipeline, not a generic list-processing language.

Initial expression operations:

- load column, constant, captured scalar, and row visibility
- integer/date/decimal add, subtract, multiply, and comparison
- float arithmetic and comparison with explicit NaN behavior
- SQL equality, NULL tests, three-valued AND/OR/NOT
- dictionary-code equality and membership
- selection mask AND/OR/AND-NOT
- projection into final rows or another physical storage sink
- COUNT, SUM, MIN, MAX, and AVG partial state

Later operations may add string prefix/range predicates, hash calculation,
group lookup/update, join probing, and top-k comparison.

Each value carries at least:

```text
physical type
SQL logical type
nullable / validity source
storage encoding
borrowed or owned lifetime
scalar, shard-vector, selection, or aggregate-state shape
```

SQL three-valued logic must be represented by validity/truth masks, not by
calling scalar truth conversion for each row. Decimal scale and intermediate
width are part of the opcode. An unsupported combination rejects compilation;
it must never silently use approximate float arithmetic.

## Shard-wide execution model

One worker owns one shard execution at a time:

```text
index/RecSet candidate source
  -> visibility mask
  -> fused predicate masks
  -> projection or aggregate kernels
  -> shard-local result
  -> existing reduce2 / ordered merge
```

### Selection representation

Use two interchangeable shard-relative representations:

- dense bitmap: one bit per row; about 7.5 KiB for a 60,000-row shard;
- sparse sorted `[]uint32` record IDs, preferably borrowed from index/RecSet
  iteration or a worker-owned reusable buffer.

The executor chooses the representation from candidate density and may switch
once when measured density crosses a calibrated threshold. Repeated conversion
between bitmap and ID list is forbidden. Visibility, deletion, RecSet, and
predicate masks should be combined in place when ownership permits.

### Column access

`ColumnStorage.GetValue` is the scalar compatibility API and cannot be the hot
vector path. Add an optional capability interface rather than widening every
storage implementation immediately. Conceptually:

```go
type VectorColumn interface {
	VectorSpans(snapshot VectorSnapshot) []VectorSpan
}
```

`VectorSpan` describes borrowed encoded/native data, row range, validity, and
the supported kernels. It must not expose mutable shard internals after the
shard lock is released. The concrete API must follow the existing rights and
locking contract: acquire read rights, load stable column storages, take the
shard read lock for the visibility snapshot, and release everything with
panic-safe defers.

Useful first storage paths:

- `StorageInt` and date: operate on native or bit-packed integer data;
- decimal: operate on scaled integers without converting to `Scmer`;
- const/enum/dictionary: compare codes or one constant directly;
- float: contiguous numeric spans;
- SCMER/sparse/compute proxy/delta: scalar fallback until they expose a safe
  vector view.

The delta tail and transactional overlays may be evaluated with a separate
scalar or vector path. The main compressed column must not be decoded merely
because a small delta exists.

### No avoidable materialization

- Filters produce or refine a selection, not an array of booleans.
- Aggregates consume selected storage values directly into registers and one
  shard-local state.
- Projection converts to `Scmer` only at the API/result edge or when feeding an
  unsupported scalar downstream operator.
- Join probes consume selected keys directly; they do not first construct a
  Scheme list of keys.
- A reusable intermediate relation is written once into the selected storage
  carrier, never duplicated into a vector-owned relation.
- Scratch buffers are worker-owned and reused. Their size is a kernel detail;
  they do not introduce a second 2,048-row execution domain.

## AVX implementation

### Dispatch

Provide architecture-specific kernel tables selected once at startup:

```text
scalar reference
amd64 AVX2
amd64 AVX-512 where available and measurably beneficial
arm64 NEON later
```

Use build tags for architecture code and runtime CPU feature detection for the
amd64 level. Plans and cached programs name semantic operations and required
capabilities, never concrete register numbers.

### Kernel strategy

Start with precompiled kernels, preferably assembly for loops the Go compiler
does not vectorize reliably:

- bit-packed/native integer comparison -> mask
- integer/decimal SUM and COUNT over mask
- dictionary-code equality/IN -> mask
- bitmap combination and population count
- numeric projection/copy into a physical sink
- hash generation and batched hash probe later

Each kernel processes the complete storage span by advancing SIMD registers.
Scalar prefix/tail handling is part of the kernel. AVX-512 masked loads may be
used when profitable; AVX2 uses explicit bitmap/ID iteration. Do not use gather
for dense data when sequential loads are possible. For sparse candidates,
benchmark scalar ID iteration against AVX gather rather than assuming gather is
faster.

The existing scalar JIT and `Declaration.JITEmit` can later gain a vector
equivalent, for example `VectorEmit`. That second stage may fuse several IR
operations into one query-specific loop. It must share the typed IR, validity
rules, feature dispatch, fingerprinting, compile budget, and scalar fallback
with the precompiled backend.

## `scan_order`, indexes, LIMIT, and braking

Vectorization must not destroy ordering to obtain a dense scan.

- An ordered index remains the owner of row order.
- OFFSET/LIMIT and partition limits remain scan boundaries.
- Early braking wins over vectorization when only a few ordered rows are
  required.
- Vector predicates may filter an ordered candidate ID stream, but results are
  emitted in the original stream order.
- Dense monotonic index ranges may be exposed as spans and vectorized.
- Irregular ordered IDs use sparse evaluation or scalar fallback.
- Sorting/top-k is a separate physical operator and should consume typed column
  values without first materializing Scheme rows.

`scan_order` therefore needs the same vector compiler, but its cost threshold
will usually be higher than for a full `scan`. The currently disabled batch
rewrite is not enabled merely by adding vectors; reduce2, final partial-batch,
outer-row, ordering, and LIMIT semantics must first be represented explicitly.

## Joins and grouping

The first milestone vectorizes leaf scans and scalar aggregates. It already
removes substantial callback and boxing cost without redesigning joins.

Next milestones extend the same pipeline:

- vector hash build from selected key spans;
- batched hash/index probes with a result mask or match-ID vector;
- dictionary-code joins when dictionary identity or translation is proven;
- vector group-key hashing and shard-local aggregate tables;
- RecSet projection directly from match masks;
- merge of shard-local aggregate states through the existing reduce2 ownership
  contract.

The logical join tree remains authoritative. Vector execution changes how one
physical node consumes its children, never which aliases are joined or where an
outer-join barrier sits.

## Cost model and plan choice

Vectorization is not unconditionally faster. Add measured physical facts:

```text
vector_startup_ns
kernel_ns_per_row by operation/type/encoding/CPU level
selection_bitmap_ns_per_word
sparse_probe_ns_per_id
decode_ns_per_row
scalar_fallback_ns_per_row
result_box_ns_per_row
vector_compile_ns
scratch_bytes
```

The cost calculation must include expected candidate density, output rows,
LIMIT braking, ordering, storage encoding, cold/warm state, reuse, and the
number of unsupported scalar islands. A pipeline with a scalar call per passing
row may still win when the filter is vectorized and highly selective. A
pipeline with a scalar call per input row normally should not be selected.

`EXPLAIN` should show:

- scalar or vector execution;
- selected CPU kernel level;
- supported fused operations and scalar islands;
- dense bitmap or sparse-ID strategy;
- estimated rows/density and scalar/vector cost;
- fallback reason when vector compilation was rejected.

## Correctness and fallback contract

Every `VecProgram` contains or references the original optimized scalar plan.
Fallback may happen at compile time for unsupported IR or at execution startup
for an incompatible storage/CPU capability. Mid-shard fallback is allowed only
before externally visible output or side effects and must restart that shard
from a clean accumulator.

Differential tests execute scalar and vector paths on identical snapshots and
compare:

- values, order, multiplicity, and NULLs;
- empty input and outer-row behavior;
- decimal scales, overflow, NaN, and signed zero;
- main column plus inserts/deletions and each transaction mode;
- dense scan, range, RecSet, unique point, and ordered LIMIT;
- shard boundary sizes including 0, 1, SIMD width +/- 1, and shard size +/- 1;
- scalar fallback islands and unsupported storage encodings.

No vector optimization is accepted merely because TPC-H output matches once.
Repeated execution and plan-cache variants must be included to prevent a repeat
of cached aggregate ownership/correctness failures.

## Delivery packages

### 1. Baseline and vector IR

- Record scalar CPU/allocation profiles for Q1, Q6, Q12, and representative ERP
  point/range/report queries.
- Define typed vector IR, fingerprints, capability errors, and scalar IR
  interpreter.
- Add declaration metadata for pure/vectorizable operations and reducers.

Acceptance: scalar IR is bit-for-bit equivalent to existing callbacks; compile
time and IR size are bounded and reported.

### 2. Shard selection and raw numeric views

- Add worker-owned shard bitmap/sparse-ID abstractions.
- Add safe vector spans for integer, date, decimal, const, enum/dictionary, and
  float storages.
- Keep delta/compute-proxy fallback explicit.

Acceptance: selection construction has no allocation per input row and obeys
all shard locking/visibility rules.

### 3. Precompiled vector scan kernels

- Implement scalar, AVX2, and optional AVX-512 dispatch.
- Fuse numeric/dictionary filters with COUNT/SUM/MIN/MAX/AVG.
- Convert to `Scmer` only for final output.

Acceptance: supported aggregate scans have zero allocations per input row,
match scalar results in differential tests, and show a measured speedup over
the scalar path on SF0.1 and SF1.

### 4. Physical lowering (A)

- Add vector candidates to `build_queryplan` only.
- Connect candidates to the common physical cost domain and guarded plan cache.
- Add `EXPLAIN` components and scalar fallback reasons.

Acceptance: no vector artifact appears before physical lowering; cost guards
select scalar plans for point lookups/early LIMIT where they are faster.

### 5. Generic scan specialization (B)

- Extend `scan` and then `scan_order` optimizer hooks to compile the same IR.
- Reuse optimizer rewrite budgets, fingerprints, type/ownership facts, and
  precondition reporting.

Acceptance: hand-written and legacy scans obtain the same vector program as an
equivalent physical SQL candidate, without recursive hook loops or AST growth.

### 6. Vector joins, grouping, and sinks

- Add batched hash/index probes, vector group updates, typed top-k/sort input,
  and direct storage-carrier writes.
- Add optional vector JIT only after stable precompiled kernels establish the
  semantics and cost baseline.

Acceptance: targeted TPC-H problem shapes and generic ERP reports improve
without regressing point/range scans, compile time, transactional correctness,
or memory bounds.

## Recommended implementation order

Implement **A+B on one backend**, but land the backend and A first:

1. typed IR plus scalar reference;
2. shard masks and borrowed storage spans;
3. AVX numeric filter/aggregate kernels;
4. cost-based physical SQL lowering;
5. generic `scan` hook recognition;
6. ordered scans, joins, grouping, strings, and optional vector JIT.

This preserves planner ownership, gives SQL plans an intentional cost-based
choice, and still lets generic Scheme/storage plans benefit. It also avoids the
two failure modes of the isolated variants: A alone would leave many existing
scan callbacks scalar, while B alone would hide a major physical strategy from
the planner and its cost model.
