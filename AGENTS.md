# Repository Guidelines

## Project Structure & Modules
- `main.go`: entrypoint and CLI flags.
- `scm/`: Scheme runtime (REPL, HTTP/MySQL servers, builtins).
- `storage/`: core engine (tables, shards, indexes, persistence).
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
- Scheme (`.scm`): follow existing patterns; filenames are kebab-case; keep code self-explanatory with minimal comments.
- Tests: YAML files use lower_snake_case keys and `NN_description.yaml` naming.
- Avoid introducing new tools; prefer editing existing files over adding new ones.

## Testing Guidelines
- Framework: YAML specs executed via `run_sql_tests.py` against HTTP endpoints.
- Expectations: use `expect: { rows: N, data: [...] }` or `expect: { error: true }`.
- One statement per test case. Use `setup: []`/`cleanup: []` unless needed.
- No explicit coverage threshold; add tests for every feature/bugfix.

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
