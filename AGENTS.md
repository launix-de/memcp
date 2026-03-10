# Repository Guidelines

## Project Structure & Modules
- `main.go`: entrypoint and CLI flags.
- `scm/`: Scheme runtime (REPL, HTTP/MySQL servers, builtins).
- `storage/`: core storage engine (tables, shards, indexes, persistence).
- `lib/`: Scheme modules; SQL parsers and planner live here (`lib/sql-parser.scm`, `lib/psql-parser.scm`, `lib/queryplan.scm`); `lib/main.scm` bootstraps the API.
- `tests/`: YAML suites named `NN_description.yaml` (e.g., `01_basic_sql.yaml`).
- `run_sql_tests.py`: test runner (HTTP-based); starts `memcp` when needed.
- `docs/`: generated API/reference docs.
- if you work on files, update/add the copyright notice's current year

## Development Workflow

Every change — bugfix, feature, refactor — must go through a **branch + PR** cycle. Direct commits to `master` are blocked by GitHub branch protection.

1. **Create a worktree + branch** for your change:
   ```
   git worktree add ../memcp-my-feature -b my-feature
   cd ../memcp-my-feature
   ```
2. **Develop and commit** on the branch. The pre-commit hook runs the full test suite before each commit.
3. **Open a PR** targeting `master`:
   ```
   gh pr create --title "Short description" --body "What and why"
   ```
4. **CI must be green** (`test` GitHub Actions job) before the PR can be merged.
5. **Merge the PR** on GitHub (or `gh pr merge`). Delete the branch and worktree afterwards.

`git commit --no-verify` is allowed for intermediate commits on non-`master` branches, after a successful `make test` run in the current worktree/session to save iteration time, or for documentation-only changes (e.g. `README*`, docs, manuals) that do not modify executable code or tests. Before pushing/opening a PR, run the full hook-equivalent test suite (e.g. `make test`) again for any change set that includes code/test changes.

## Build, Test, and Dev Commands
- Build: `go build -o memcp` or `make` (default builds).
- Run: `./memcp --api-port=4321 lib/main.scm` (MySQL off by default in tests).
- Background run: always use `--no-repl` when running memcp in the background, otherwise closing stdin causes the REPL to exit: `./memcp --no-repl --api-port=4321 lib/main.scm &`
- Quick test (single file): `python3 run_sql_tests.py tests/01_basic_sql.yaml`.
- Connect-only (reuse a running instance): `python3 run_sql_tests.py tests/02_functions.yaml 4321 --connect-only`.
- Pre-commit: `git commit` runs all `tests/[0-9][0-9]_*.yaml` via a single `memcp` instance (port 4400). Bypassing with `--no-verify` is permitted for intermediate commits on non-`master` branches, after a successful `make test` run, or for documentation-only changes (README/docs/manuals) without code/test modifications; still run the full suite before pushing whenever code/test changes are part of the branch.

### Exact Server Invocation (used by test runner)
The test runner (`run_sql_tests.py`) starts the server exactly like this:
```
./memcp -data /tmp/memcp-sql-tests-PORT --api-port=PORT --mysql-port=PORT+1000 --disable-mysql lib/main.scm
```
- The binary MUST be `./memcp` (hardcoded in the test runner).
- `-data DIR` sets the data directory (positional dash flag, not `--datadir`).
- `--api-port=PORT` sets the HTTP API port (default 4321).
- `--no-repl` for background daemon use (test runner uses stdin pipe instead).

### Manual Server Testing
To start a server manually for debugging:
```
mkdir -p /tmp/memcp-manual-test
./memcp -data /tmp/memcp-manual-test --api-port=4399 --no-repl lib/main.scm </dev/null &>/tmp/memcp_manual.log &
```
Then send queries via the HTTP API:
```
curl -s -u root:admin "http://localhost:4399/sql/DBNAME" -d "SELECT 1"
```
- Auth: HTTP Basic `root:admin` (default password for fresh data dirs).
- API endpoint: `/sql/<database-name>` (NOT `/schema` or bare root).
- Database is auto-created on first use.
- Tests use database name `memcp-tests` on port 4321 (or custom port).

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
- Use a functional programming style. Use set/define only for function declaration or single state assignments, prefer the match coding patterns over iterative approaches.
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
- the queryplan builder is basically a code generator. code is represented as data in scheme. to create the expression (+ 4 5) you have to quote everything whose computation should be delayed: '('+ 4 5) will evaluate 4 and 5 immediately and delay the addition
- to produce functions, you must quote the symbols and the lists, e.g. '((quote lambda) '('param1 'param2) ...) or '('lambda '('param1 'param2)  ...), ' is the quoting character. '(...) constructs a list while (x) will execute the function x immediately. ('x) is invalid because symbols cannot be executed.
- constructed code can be executed via (eval ...)

## Testing Guidelines
- Framework: YAML specs executed via `run_sql_tests.py` against HTTP endpoints.
- Create one YAML file per topic; prefer adding new tests to existing files instead of adding files
- Expectations: use `expect: { rows: N, data: [...] }` or `expect: { error: true }`.
- One statement per test case. Use `setup: []`/`cleanup: []` unless needed.
- Make sure you drop and create all needed tables in setup and drop cleanup
- Even when using qualified names (schema.table) never touch tables outside of `memcp-tests`
- No explicit coverage threshold; add tests for every feature/bugfix.
- Don't forget to also add must-fail tests with expect: { error: true }

### Coverage Measurement
- Build with coverage: `go build -cover -o memcp -covermode=atomic`
- Run with coverage: `GOCOVERDIR=/tmp/memcp-coverage ./memcp ...`
- The `make test` / `git-pre-commit` script supports `MEMCP_COVERAGE=1 MEMCP_COVERDIR=/tmp/memcp-coverage` environment variables to automatically build with coverage, collect profiles, and report.
- Generate report: `go tool covdata textfmt -i=/tmp/memcp-coverage -o=/tmp/memcp-coverage/coverage.out && go tool cover -func=/tmp/memcp-coverage/coverage.out`
- HTML report: `go tool cover -html=/tmp/memcp-coverage/coverage.out`
- Current coverage: **69.7%** (594 Scheme unit tests in `lib/test.scm`, 72 storage type tests + ~2470 SQL tests in `tests/*.yaml`)
- Two test layers:
  - `lib/test.scm`: Scheme-level unit tests for the runtime (match, compare, date, strings, lists, scheduler, sync, optimizer, JIT). These run at startup before the SQL frontend.
  - `tests/*.yaml`: SQL integration tests via HTTP API. Cover SQL parsing, query planning, storage compression, triggers, persistence, DML/DDL.
- To bypass static type validation in Scheme tests (e.g., passing an int to a function declared as accepting string), wrap with an identity function: `(define _i (lambda (x) x))` then `(parse_date (_i 1718451045))`.

## Debugging Strategy
- run memcp with enabled mysql frontend on port 3307, make sure to capture its output and enable TracePrint in settings
- run some unit tests or let the user click through the software
- read through the captured output, identify the failing queries and inspect the caught panics
- construct a test case in tests/ that will reproduce the failing bug, verify bug reproduction with make test
- work on the bugfix
- verify with make test, reiterate
- if any regression appears in `make test`, stop immediately and fix the root cause before continuing any other work
- do not commit until all tests including the new one pass

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

## Performance Benchmarking & A/B Testing
- Go microbenchmarks live next to the code under test: `storage/storage-enum_test.go`, etc.
- Run benchmarks: `go test ./storage/ -bench 'BenchmarkEnumPerElem' -run '^$' -benchtime=1s`
- For A/B comparisons, build two binaries and use `benchstat`:
  ```
  git stash && go test ./storage/ -bench '...' -count=5 > /tmp/bench_A.txt
  git stash pop && go test ./storage/ -bench '...' -count=5 > /tmp/bench_B.txt
  benchstat /tmp/bench_A.txt /tmp/bench_B.txt
  ```
- When adding a `--binary` flag or build tag for alternative code paths, test both variants under the same benchmark harness to compare.

## Data Safety Policy

Preventing user-visible data loss is a first-class requirement. Every code
change touching storage, DDL, persistence, or cleanup must comply with these
policies. See also `docs/engine-semantics.md` for the user-facing reference.

### ENGINE Durability Contract

| Engine  | Durability guarantee | Crash recovery | Notes |
|---------|----------------------|----------------|-------|
| `safe`  | Full ACID, power-outage safe | WAL fsync'd + replayed at startup | **Default.** Use for all production data. |
| `logged`| Process-crash safe (no fsync) | WAL written (no fsync) + replayed | Use when fsync latency is unacceptable and hardware power protection is provided. |
| `sloppy`| Columnar files on disk, **no WAL** | In-memory deltas since last rebuild are **lost** | Use only for reconstructible data or caches. |
| `memory`| None — RAM only | **All data lost on restart** | Never for data that must survive a restart. |

### ENGINE Transition Rules

- **Any persisted → `memory`**: IRREVERSIBLE. All on-disk column files and WAL
  are deleted immediately upon `ALTER TABLE … ENGINE=memory`. No undo. Ensure
  data is either backed up or no longer needed before issuing this statement.
- **`safe`/`logged` → `sloppy`**: WAL is closed and removed. Deltas since the
  last rebuild become crash-unsafe going forward.
- **`memory` → any persisted**: Safe — current in-RAM data is serialised to disk.
- **`sloppy` → `safe`/`logged`**: Safe — WAL is opened; future writes are durable.
- **`safe` ↔ `logged`**: No-op at the shard level; both use WAL identically.

### Cleanup Code Contract

- **`shardCleanup()` (LRU eviction)**: MUST NEVER delete persistent data. It
  only releases in-memory column representations; disk files remain intact.
  Eviction of Memory-engine shards is unconditionally blocked (data would be
  permanently lost). Eviction of shards with unflushed deltas is blocked.
- **`RemoveFromDisk()`**: Called ONLY from `DropTable` (explicit user DDL) or
  from the `transitionShardEngine` Persisted→Memory path (explicit ALTER TABLE).
  Protected by `sync.Once` to prevent double-deletion.
- **`removePersistence()`**: Even more targeted — exclusively called from the
  Persisted→Memory branch of `transitionShardEngine`. Must NEVER be added to
  background or eviction code paths.
- **Trigger callbacks (`AfterDropTable`, etc.)**: MUST NOT delete data in
  unrelated tables or shards, except through explicitly declared CASCADE foreign
  key policies configured by the user. Triggers are lifecycle hooks, not GC.

### Serialization Format Stability Contract

See the versioning rules in the `storages` map comment in `storage/storage.go`.
Summary:
1. Magic bytes are permanent assignments — never change or reassign.
2. Old `deserializeXxxVN` helpers must never be deleted; on-disk data must stay
   readable forever.
3. When changing binary layout: increment the version constant, add a new helper,
   leave the old one untouched.
4. Magic bytes 1, 2, 13, 40 (legacy, no version byte): if their format ever
   needs to change, register a NEW magic byte; keep the old one as a read-only
   legacy reader.

### Implementation Checklist for Storage Changes

Before merging any PR that touches storage, persistence, DDL, or cleanup code:
- [ ] No new code path calls `removePersistence()` or `RemoveFromDisk()` outside
      the approved call sites above.
- [ ] No eviction or background cleanup path can reach disk-deletion code.
- [ ] Any new binary format bump follows the magic/version byte rules.
- [ ] ENGINE semantics table above remains accurate (update if behaviour changes).
- [ ] `AfterDropTable` / trigger callbacks reviewed for unintended side-effects.

## Release Process

### Version Numbering
- Scheme: `MAJOR.MINOR` — no dates in the version string.
- `0.x` while the API is still evolving; bump to `1.0` for the first stable API.
- Bump `MINOR` for every normal release; bump `MAJOR` only for breaking API changes.

### CHANGELOG.md Format
The first line of `CHANGELOG.md` must be the bare version number followed by an em-dash and ISO date:
```
0.2 — 2026-03-XX
=================
- feat: short description
- fix: short description
```
The Makefile reads `VERSION` from the first word of that line (`awk '{print $1}'`).

### Release Steps
1. Add a new entry at the top of `CHANGELOG.md` with the new version and date.
2. Commit the changelog update to `master` (via PR as usual).
3. Tag the release commit:
   ```
   git tag -a v0.2 -m "Release 0.2"
   git push origin v0.2
   ```
4. Build artefacts:
   ```
   make memcp_0.2_amd64.deb memcp_0.2_x86_64.rpm
   ```
   Output filenames include version and arch automatically (e.g. `memcp_0.2_amd64.deb`, `memcp_0.2_x86_64.rpm`).
5. Create GitHub release with artefacts:
   ```
   gh release create v0.2 memcp_0.2_amd64.deb memcp_0.2_x86_64.rpm \
     --title "v0.2" --notes-file <(sed -n '/^0\.2/,/^[0-9]/p' CHANGELOG.md | head -n -2)
   ```
6. Push Docker image (requires `docker login` once):
   ```
   make docker-release
   ```
   This tags `carli2/memcp:0.2` **and** `carli2/memcp:latest` and pushes both.

### Makefile Variables
- `VERSION` — auto-read from `CHANGELOG.md`; override with `make VERSION=0.2 memcp_0.2_amd64.deb`.
- `DEB_ARCH` — defaults to `dpkg --print-architecture` (e.g. `amd64`).
- `RPM_ARCH` — defaults to `uname -m` (e.g. `x86_64`).

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
