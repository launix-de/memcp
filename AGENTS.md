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
- Quick test (single file): `python3 run_sql_tests.py tests/01_basic_sql.yaml`.
- Connect-only (reuse a running instance): `python3 run_sql_tests.py tests/02_functions.yaml 4321 --connect-only`.
- Pre-commit: `git commit` runs all `tests/[0-9][0-9]_*.yaml` via a single `memcp` instance (port 4400). Bypass only if necessary: `git commit --no-verify -m "..."`.

## Coding Style & Naming
- in every source file: include license header
- Go: format with `go fmt ./...` after each patch; keep idiomatic Go (short, cohesive funcs). Tabs and gofmt defaults are expected.
- Go imports: sort imports by path length ascending within each import block.
- Scheme (`.scm`): follow existing patterns; filenames are kebab-case; keep code self-explanatory, add comments for more complex code parts.
- Please remember, you are a textmodel and you are not trained to functional programming languages like scheme. Don't correct braces at your will, try to use tools like sed, run it only on small portions of the code
- Indentation in Scheme: indent nested forms with tabs; align continuation lines to opening form where helpful; avoid mixing tabs and spaces.
  - After touching any files under `lib/`, run the Scheme formatter to normalize indentation and catch bracket issues:
    - Reformat in-place: `python3 tools/lint_scm.py`
    - Check only (CI-friendly): `python3 tools/lint_scm.py --check`
  - The formatter indents with tabs based on parenthesis depth and warns if parentheses are unbalanced or depth goes negative on any line.
- In this scheme dialect (set var value) does not change any value of the outer scope - it rather works exactly like (define )
- lambdas in scheme have only two parameters: (paramlist) and body: `(lambda (a b) (+ a b))`. if you want to compute multiple statements inside the body, you must open a (begin) block
- Tests: YAML files use lower_snake_case keys and `NN_description.yaml` naming.
- Avoid introducing new tools; prefer editing existing files over adding new ones.

### API Stability
- Avoid "Ex"-suffixed helper variants (e.g., `FooEx`); prefer a single, clear API that accepts parameters like `alreadyLocked bool` when needed.
- When changing function signatures, update all call sites in the repository in one pass; do not leave temporary wrappers.

### Concurrency Rules (Storage Engine)
- Never access shard internals without the shard lock:
  - `storageShard.columns`, `deltaColumns`, `inserts`, `deletions`, and `Indexes` must only be read/written while holding `t.mu`.
  - Use `RLock` for read-only snapshots and `Lock` for mutations. Do not read Go maps without a lock.
  - Before touching a shard, acquire concurrency rights with GetRead() or GetWrite(). Make sure you release all locks and rights with a proper panic-safe defer
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

### Scheme AST and Codegen Quoting (lib/queryplan.scm)
- Build AST as data: most builder blocks use a single leading quote `'(...)` so nested lists are data, not executed at construction.
- Lambdas: embed as `'((quote lambda) (param-list) body)` where:
  - `param-list`: a list of (quoted) or 'quoted symbols. Follow existing patterns like `(map cols (lambda (col) (symbol (concat tblvar "." col))))` rather than constructing `(cons 'list ...)` for params.
  - `body`: if you want to produce a list value at runtime, construct it explicitly with `(cons (quote list) ...)` instead of a bare literal like `(1 "x")` which would be treated as a call.
- Reduce/reduce2: quote bodies so they are embedded correctly, e.g. `'(set_assoc acc rowvals true)` inside `'('lambda '('acc 'rowvals) ...)`.
- Quote operator symbols, not variables: keep identifiers like `schema`, `grouptbl`, and `rows` unquoted so they bind at querybuilder runtime; quote only procedure names and constructors inside quoted ASTs as well as your own lambda definitions.
- Reducers: embed bodies quoted, e.g. `'('set_assoc 'acc 'rowvals true)` so set_assoc is emitted, not executed at build time.
- Neutral values: choose a proper literal, e.g. `'(list)` for assoc reducers; avoid `nil` when callee expects a list/FastDict.
- Runtime tuples: construct with `(list ...)` using bound symbols, not `'(sym1 sym2)` — the latter is a literal tuple and collapses all rows to one key.
- Column-name lists: functions like `insert` expecting column names take quoted string lists, e.g. `'("col1" "col2")`.

### Functional Semantics
- This Scheme dialect is purely functional; procedures have no side effects and are fully parallelizable.
- `(set symbol value)` is scope-local and works like (define ). there is no real set
- To communicate outside scope, use sessions via `(newsession)` which are threadsafe.

## Testing Guidelines
- Framework: YAML specs executed via `run_sql_tests.py` against HTTP endpoints.
- Create one YAML file per topic; prefer adding new tests to existing files instead of adding files
- Expectations: use `expect: { rows: N, data: [...] }` or `expect: { error: true }`.
- One statement per test case. Use `setup: []`/`cleanup: []` unless needed.
- Make sure you drop and create all needed tables in setup and drop cleanup
- Even when using qualified names (schema.table) never touch tables outside of `memcp-tests`
- No explicit coverage threshold; add tests for every feature/bugfix.
- Don't forget to also add must-fail tests with expect: { error: true }

## Security & Configuration
- Do not expose development instances publicly. Use `--api-port`, `--mysql-port`, `--disable-mysql`, and `-data <path>` to isolate local runs.
- Large datasets should not be added to Git; use local paths under `data/`.

## Execution Plan Building Hints (Pipelines & Braking)
- Operator pipelines: fuse filter → project → aggregate in a single scan to reduce overhead
- Late materialization: read only referenced columns; assemble rows at the edge (joins/output), not in the hot path.
- Range-based braking: for ORDER BY + LIMIT, maintain a top-k threshold and stop scanning when next-best keys cannot beat it; prefer braking over inner materialization if orders are compatible.
- Join pipelines: drive ordered/filtered side; hash/range-probe the other; keep fuse-friendly structure; precompute hidden computed columns for FK/PK group reuse.

## Memory & CPU Efficiency
- Design principle: Cache misses are more expensive than lightweight compression. Prefer compact encodings (e.g., bit-packing 3/5‑bit integers) and sequential scans over scattered, cache‑cold access.
- Use columnar storage to keep footprints small and hot; compress where it reduces cache lines touched even if it adds tiny (de)compression overhead.
- Pull function calls out of loops whenever possible
- Always operate on batches of data
- Do not call goroutines on small data batches or single-item loops
- Avoid lock/unlock inside loops

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
