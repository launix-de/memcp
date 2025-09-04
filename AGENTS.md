# Repository Guidelines

## Project Structure & Modules
- `main.go`: entrypoint and CLI flags.
- `scm/`: Scheme runtime (REPL, HTTP/MySQL servers, builtins).
- `storage/`: core storage engine (tables, shards, indexes, persistence).
- `lib/`: Scheme modules; SQL parsers and planner live here (`lib/sql-parser.scm`, `lib/psql-parser.scm`, `lib/queryplan.scm`); `lib/main.scm` bootstraps the API.
- `tests/`: YAML suites named `NN_description.yaml` (e.g., `01_basic_sql.yaml`).
- `run_sql_tests.py`: test runner (HTTP-based); starts `memcp` when needed.
- `docs/`: generated API/reference docs.

## Build, Test, and Dev Commands
- Build: `go build -o memcp` or `make` (default builds).
- Run: `./memcp --api-port=4321 lib/main.scm` (MySQL off by default in tests).
- Quick test (single file): `python3 run_sql_tests.py tests/01_basic_sql.yaml 4400`.
- Connect-only (reuse a running instance): `python3 run_sql_tests.py tests/02_functions.yaml 4400 --connect-only`.
- Pre-commit: `git commit` runs all `tests/[0-9][0-9]_*.yaml` via a single `memcp` instance (port 4400). Bypass only if necessary: `git commit --no-verify -m "..."`.

## Coding Style & Naming
- Go: format with `go fmt ./...`; keep idiomatic Go (short, cohesive funcs). Tabs and gofmt defaults are expected.
- Go imports: sort imports by path length ascending within each import block.
- Scheme (`.scm`): follow existing patterns; filenames are kebab-case; keep code self-explanatory, add comments for more complex code parts.
- Indentation:
  - Go: use tabs; do not mix spaces; rely on gofmt to normalize.
  - Scheme: indent nested forms with tabs; align continuation lines to opening form where helpful; avoid mixing tabs and spaces.
- Tests: YAML files use lower_snake_case keys and `NN_description.yaml` naming.
- Avoid introducing new tools; prefer editing existing files over adding new ones.

### Concurrency Rules (Storage Engine)
- Never access shard internals without the shard lock:
  - `storageShard.columns`, `deltaColumns`, `inserts`, `deletions`, and `Indexes` must only be read/written while holding `t.mu`.
  - Use `RLock` for read-only snapshots and `Lock` for mutations. Do not read Go maps without a lock.
- Avoid lock upgrades. Do not acquire `t.mu.Lock()` while holding `t.mu.RLock()`.
  - Pattern for lazy-load under concurrency:
    1. `RLock` → check if value is present; `RUnlock`.
    2. If missing, `Lock` → re-check → compute/store → `Unlock`.
- Prefer helper APIs that encapsulate locking:
  - Use `getColumnStorageOrPanic(name)` to obtain a stable `ColumnStorage` pointer (loads on demand) without racing writers.
  - Use `ColumnReader(name)` rather than reading `t.columns[name]` directly.
- Scan/plan code must not read from `t.columns[...]` directly. Fetch storages with helpers outside of long-held locks; then take `RLock` only for index iteration and reading `inserts`/`deletions`/`deltaColumns`.
- Log replay and rebuild mutate shard state and must hold `t.mu.Lock()` for their critical sections. They must not take table locks inside shard locks to avoid cycles.
- When adding new storage fields, document the locking discipline and update this section.

### Scheme AST Quoting (lib/queryplan.scm)
- Build AST as data: most builder blocks use a single leading quote `'(...)` so nested lists are data, not executed at construction.
- Lambdas: embed as `'((quote lambda) (param-list) body)` where:
  - `param-list`: a list of (quoted) symbols. Follow existing patterns like `(map cols (lambda (col) (symbol (concat tblvar "." col))))` rather than constructing `(cons 'list ...)` for params.
  - `body`: if you want to produce a list value at runtime, construct it explicitly with `(cons (quote list) ...)` instead of a bare literal like `(1 "x")` which would be treated as a call.
- Reduce/reduce2: quote bodies so they are embedded correctly, e.g. `'(set_assoc acc rowvals true)` inside `'((quote lambda) '('acc 'rowvals) ...)`.
- Quote operator symbols, not variables: keep identifiers like `schema`, `grouptbl`, and `rows` unquoted so they bind at runtime; quote only procedure names and constructors inside quoted ASTs as well as your own lambda definitions.

### Codegen Quoting Patterns
- Lambda params: always emit a quoted list of symbols, e.g. `'((quote lambda) '(acc rowvals) body)`. Do not synthesize params from strings; never leave them as `(nil nil)`.
- Reducers: embed bodies quoted, e.g. `'(set_assoc acc rowvals true)` so set_assoc is emitted, not executed at build time.
- Neutral values: choose a proper literal, e.g. `'(list)` for assoc reducers; avoid `nil` when callee expects a list/FastDict.
- Runtime tuples: construct with `(list ...)` using bound symbols, not `'(sym1 sym2)` — the latter is a literal tuple and collapses all rows to one key.
- Column-name lists: functions like `insert` expecting column names take quoted string lists, e.g. `'("col1" "col2")`.
- Outer refs and variables: leave `schema`, `grouptbl`, and `(outer ...)` unquoted inside emitted code to bind at evaluation time; quote only syntactic operators.

### Functional Semantics
- This Scheme dialect is purely functional; procedures have no side effects and are fully parallelizable.
- `(set symbol value)` is scope-local. To communicate outside scope, use sessions via `(newsession)` which are threadsafe.

## Testing Guidelines
- Framework: YAML specs executed via `run_sql_tests.py` against HTTP endpoints.
- Create one YAML file per topic; prefer adding new tests to existing files instead of adding files
- Expectations: use `expect: { rows: N, data: [...] }` or `expect: { error: true }`.
- One statement per test case. Use `setup: []`/`cleanup: []` unless needed.
- No explicit coverage threshold; add tests for every feature/bugfix.
- Don't forget to also add must-fail tests with expect: { error: true }

## Commit & Pull Requests
- Commits: small, incremental, and descriptive (what/why). Run tests before pushing. If an agent co-authored changes, include a `Co-Authored-By:` trailer.
- PRs: clear description, linked issues, steps to reproduce, and before/after benchmarks for performance work. Include new/updated tests and notes on flags used (e.g., `--api-port`, `--disable-mysql`).

## Security & Configuration
- Do not expose development instances publicly. Use `--api-port`, `--mysql-port`, `--disable-mysql`, and `-data <path>` to isolate local runs.
- Large datasets should not be added to Git; use local paths under `data/`.

## Unnesting Strategy
- Principle: avoid materialization whenever possible. Nested `SELECT` are flattened as a rename of columns into a single FROM tableset.
- Materialize only when required (e.g., inner `GROUP BY/HAVING`, `LIMIT/OFFSET`, or correlation). Use hidden temp tables prefixed with `.` containing grouped column names.
- FK/PK optimization: when grouping by a foreign key that references a primary key, do not create a temp table—add a hidden computed column (prefixed with `.`) on the primary-key table and compute there.

## Execution Hints (Pipelines & Braking)
- Operator pipelines: fuse filter → project → aggregate in a single scan to reduce overhead (vectorize where possible).
- Selection vectors: evaluate predicates in batches, compact indices, then project/aggregate only selected rows.
- Late materialization: read only referenced columns; assemble rows at the edge (joins/output), not in the hot path.
- Range-based braking: for ORDER BY + LIMIT, maintain a top-k threshold and stop scanning when next-best keys cannot beat it; prefer braking over inner materialization if orders are compatible.
- Join pipelines: drive ordered/filtered side; hash/range-probe the other; keep fuse-friendly structure; precompute hidden computed columns for FK/PK group reuse.

### Parallel Braking Plan
- Goal: early-stop inside shard workers, not only during global merge.
- Option 1 (preferred): per-shard ordered iteration (iterateIndexSorted) that streams tuples in ORDER BY sequence; workers read a global k-th threshold (atomic) and stop when their next-best key can’t beat it.
- Option 2 (interim): per-shard local top-k heap while scanning unsorted; publish/prune against a shared threshold; sort only the local top-k afterwards.
- Planner: propagate k = offset + limit when inner ORDER is compatible with outer ORDER so braking replaces inner materialization.

## Memory & CPU Efficiency
- Design principle: Cache misses are more expensive than lightweight compression. Prefer compact encodings (e.g., bit-packing 3/5‑bit integers) and sequential scans over scattered, cache‑cold access.
- Use columnar storage and vectorized compute to keep footprints small and hot; compress where it reduces cache lines touched even if it adds tiny (de)compression overhead.

## Optimizer Roadmap (scm/optimizer.go)
- Inline Use-Once: inline variables defined in a begin when used < 2× and not arrays; already partially present — re-enable with safety checks.
- Elide set/define: where a symbol is set once and consumed once, replace the read with the value and drop the set.
- Numbered Params: reactivate lambda parameter indexing (NthLocalVar) once nested-scope bugs are fixed; enables faster param access.
- In-place Variants: add map_inplace/filter_inplace and let optimizer switch when the first argument is disposable.
- Producer Pipelines: introduce secret pure imperative versions of functions like map, produce etc. that are based on loops and change variables; lower pure functional code to imperative one
- Cons/Merge→Append: normalize construction patterns to append for fewer allocations.
- Currying/Specialization: partial-evaluate functions with const masks to generate specialized lambdas.
- Prefetch/outer: safely replace env lookups with prebound values when no shadowing occurs; propagate (outer ...) through subtrees.
- Parser Optimizations: number parser parameters and precompile eligible patterns via parseSyntax.
- JIT: chain together scanner loop (main+delta), map function and reduce function
- JIT: specialize functions like "+" according to lhs and rhs operator types, remove type checks and so on
- JIT: inspect the inner logic of getValue on compressed columns such that some operators like (equal??) can directly operate on compressed data, e.g. for BLOB storages on the hash or on integer storages: rather compress the equal?? value that we want to search instead of decompressing the whole table

## Query plan roadmap

- **TODO:** This document outlines a strategic roadmap for integrating a learned query optimizer into our database engine, `memcp`. The goal is to replace or augment the traditional heuristic-based join reordering mechanism with a deep learning model, likely a Transformer, to achieve higher performance and adaptivity. The core idea is to learn optimal join orderings from query patterns and data statistics, moving towards a "hands-free" optimizer. [7]

### Phase 1: Foundational Data Collection and Representation

1.  **Feature Engineering & Encoding:** The first step is to define a comprehensive feature set that can represent a query and its context. This involves:
    *   **Table & Column Statistics:** Collect fundamental statistics like table cardinality, number of distinct values (NDV), and presence of indexes. [1, 26]
    *   **Predicate & Join Information:** Encode filter predicates (e.g., `WHERE a > 10`) and join predicates (`a.id = b.id`). This includes the tables, columns, operators, and constants involved.
    *   **Learned Cardinality Estimation:** The most critical feature is cardinality (the size of intermediate results). Instead of relying on traditional, often inaccurate, histogram-based estimates [26], we will implement a learned model.
        *   **Histogram Encoding with CNNs:** We will represent column data distributions (histograms) as vectors. A 1D-Convolutional Neural Network (CNN) will be trained to process these histogram vectors and produce a rich, learned embedding. [22, 27] This approach is superior to using a single, manually calculated selectivity value as it captures the entire data distribution, including skew. [21, 26]
        *   **End-to-End Training:** This "Histogram-Encoder-CNN" will be trained end-to-end with the main query planning model. This allows it to learn the most relevant features from the distribution for the specific task of predicting join costs. [3, 20]

2.  **Query Representation for the Model:** The encoded features must be structured into a format a Transformer can process.
    *   **Graph-to-Sequence:** A query can be seen as a graph where tables are nodes and joins are edges. We need to represent this graph as a sequence of vectors.
    *   **Input Tensors:** We will use two main inputs for the model:
        1.  **Table Feature Tensor `(N, F)`:** A sequence of N vectors, where each vector represents a table and its associated filters, including the learned histogram embeddings.
        2.  **Join Relation Tensor `(N, N, J)`:** An adjacency matrix-like tensor where the entry `(i, j)` is a vector describing the join between table `i` and `j` (e.g., join selectivity, join type). This explicitly provides the graph structure to the model. [15]

### Phase 2: Model Architecture and Training

1.  **Core Model: Graph-Aware Transformer:**
    *   We will implement a Transformer encoder architecture. The key innovation is to inject the `Join Relation Tensor` directly into the self-attention mechanism. This creates a "bias" in the attention scores, forcing the model to pay more attention to tables that are directly and selectively joined. This architecture is effectively a **Graph Transformer**. [15]
    *   The iterative nature of Transformer layers allows the model to simulate message passing, where each table progressively learns about its multi-hop neighborhood in the join graph. [8, 19]

2.  **Training Strategy:**
    *   **Objective:** The model's output will be a priority score (a float) for each table. The final join order is determined by sorting the tables based on these scores.
    *   **Data Generation:** We will generate a large and diverse set of training queries. For each query, we will execute multiple join permutations and record their *actual execution latency*. This provides the ground truth.
    *   **Loss Function (Learning-to-Rank):** Instead of simple regression on latency (which can be noisy), we will use a **ranking loss**. Given two join plans for the same query, the loss function will penalize the model if it assigns a better score to the slower plan. This directly optimizes for the *relative order* of plans, which is more robust. [15]
    *   **Reinforcement Learning (Future Work):** A more advanced alternative is to frame the problem as a Markov Decision Process and use Deep Reinforcement Learning (DRL). The model (agent) would build a join plan step-by-step, receiving a reward based on the final plan's cost. [7, 15] This is more complex but has shown great promise in academic research.

### Phase 3: Integration and Deployment

1.  **Inference Workflow:**
    *   When a new query arrives, the `memcp` engine will extract its features and construct the `Table Feature` and `Join Relation` tensors.
    *   These tensors are fed into the trained Transformer model for a fast forward pass.
    *   The model outputs the priority scores.
    *   The engine sorts the tables according to these scores to get the proposed join order.
    *   This order is then used to construct the final physical execution plan.

2.  **Continuous Improvement & Feedback Loop:**
    *   The system will operate in a "shadow" mode initially, comparing the learned plan's predicted cost against the heuristic planner's choice.
    *   We will continuously log query performance and feed it back into the training pipeline. This allows the model to adapt to changes in the data distribution and query workload over time, a key advantage over static optimizers. [7, 13]

### Key Research Insights & Best Practices:

*   **Learned Cardinality is Key:** The biggest source of error in traditional optimizers is inaccurate cardinality estimation. Using deep learning models (like MSCN, Naru, NeuroCard) to learn the joint data distribution is a proven way to achieve orders-of-magnitude improvements in accuracy. [4, 6, 18]
*   **Representing Plans, Not Just Queries:** The model should learn from the structure of the *plan*. Architectures like Tree-LSTMs or plan-structured neural networks are effective. Our Graph Transformer approach is a state-of-the-art implementation of this principle. [11, 29]
*   **Data-Driven vs. Query-Driven:** Our approach is a hybrid. We use data-driven models (CNNs on histograms) to learn data distributions [2] and a query-driven model (the Transformer) to learn plan structures.
*   **Start with Supervised Learning:** While Reinforcement Learning is powerful, it's harder to train and stabilize. Starting with a supervised, learning-to-rank approach is more pragmatic and provides a strong baseline. [14, 15]



## MySQL ↔ MemCP Parallel Run Plan
- Goal: operate MemCP alongside MySQL for months with minimal risk, validating correctness and performance before cutover.

### CDC Implementation Sketch
- Ingestion: connect to MySQL via replication protocol, request ROW-based binlogs with GTID enabled; handle rotate, format, table map, write/update/delete rows, and DDL events.
- Schema mapping: translate MySQL types and column sets to MemCP schemas; maintain a mapping cache keyed by (db, table, table_id, column bitmap).
- Apply pipeline: buffer event groups per transaction; apply atomically to MemCP with retries; ensure idempotency using GTID set checkpoints persisted in MemCP `system.cdc_state`.
- DDL handling: best-effort translate CREATE/ALTER/DROP to MemCP; accept FKs as metadata-only if not enforced.
- Backfill: initial snapshot via consistent `mysqldump`/`CLONE`-like read or parallel SELECTs; once complete, switch to live binlog apply.

### Cutover and Safety
- Shadow validation: run representative reads against both backends; record row diffs and latency deltas.
- Checksums: periodic `CHECKSUM TABLE`/count+sum-of-columns approximations to catch drift.
- Rollout gates: (1) schema parity validated; (2) CDC lag SLO met; (3) query diff rate below threshold; (4) error budget healthy.
- Cutover: switch writes behind a feature flag to MemCP; keep CDC in reverse (MemCP→MySQL) temporarily for fast rollback, or maintain dual-writes for a short window.

### Operational Notes
- Credentials: use a MySQL REPLICATION SLAVE user with minimal grants.
- Failure modes: on apply error, quarantine table/GTID and alert; do not silently skip.
- Observability: metrics for CDC lag, events/sec, apply errors, MemCP write latency; GTID watermark in logs and `system.cdc_state`.

## Near-Term TODOs (Adoption Focus)

- MySQL compatibility v1
  - Protocol: auth/handshake, prepared statements, multi-stmt, transactions, `affected_rows`, `last_insert_id()`.
  - Dialect/semantics: backticks, implicit casts, NULL/boolean truthiness, string escapes, date/time basics.
  - DDL essentials: CREATE/ALTER TABLE, PK/UK, indexes; accept FKs (warn/no-op).
  - UPSERT: `INSERT ... ON DUPLICATE KEY UPDATE` baseline.
  - Error codes/messages: approximate MySQL 8 for common cases.

- CDC (MySQL→MemCP)
  - Binlog client (ROW events, GTID, rotate, table map, DDL).
  - Type/charset mapping; column bitmap handling.
  - Transactional applier with idempotency; GTID checkpointing.
  - Snapshot + catch-up orchestration; observability.

- Tooling & Docs
  - 10‑minute “Run alongside MySQL” guide and cutover checklist.
  - Docker image with MySQL port enabled by default for non-test runs.
  - Compatibility harness that diffs MemCP vs MySQL for a query corpus.

## General Knowledge Highlights
- MemCP is a functional, vectorized execution engine with Scheme-driven planning; avoid side effects and prefer fused pipelines.
- SQL is parsed/planned in `lib/*.scm`; runtime/server glue is in `scm/`; storage and indexes live in `storage/`.
- Tests are YAML specs executed over HTTP; mark exploratory/compat suites `noncritical: true` until stable.
