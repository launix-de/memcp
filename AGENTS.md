<!-- Copyright (C) 2026 Carl-Philip Hänsch -->

# Repository Guidelines

## Project Structure & Modules
- `main.go`: entrypoint and CLI flags.
- `scm/`: Scheme runtime (REPL, HTTP/MySQL servers, builtins); no SQL logic allowed here
- `storage/`: core storage engine (tables, shards, indexes, persistence).
- `lib/`: Scheme modules; SQL parsers and planner live here (`lib/sql-parser.scm`, `lib/psql-parser.scm`, `lib/queryplan.scm`, `lib/queryplan-*.scm`); `lib/main.scm` bootstraps the API.
- `tests/`: SQL YAML suites organized by the taxonomy in `tests/README.md`.
- `run_sql_tests.py`: test runner (HTTP-based); starts `memcp` when needed.
- `docs/`: generated API/reference docs.
- INVARIANTS.md - must read, contains the Common Taxonomy aswell as coding rules and hardwritten architecture decisions that must not be violated
- if you work on files, update/add the copyright notice's current year

## External Dependency Policy

**WARNING: NEVER COPY, VENDOR, OR PATCH AN EXTERNAL LIBRARY INSIDE THE MEMCP
REPOSITORY.** In particular, do not add `third_party/`, `vendor/`, or another
tracked source-code copy to bypass the Go module system. Fixes belong in the
library's upstream repository, must be reviewed and tested there, and must be
published with a proper semantic version tag. MemCP may then update its normal
`go.mod` requirement to that released version. Do not commit local-path
`replace` directives. An untracked `.third_party/` directory may be used only
as disposable local workspace while preparing an upstream patch; none of its
contents may enter a MemCP commit or build.

## Human Authorization for INVARIANTS.md

`INVARIANTS.md` is the human-controlled quality and target-architecture contract.
AI agents MUST NOT edit, replace, delete, rename, or regenerate it without explicit
human authorization for the intended change. This includes additions, rewording
and formatting-only changes. General requests to implement features, fix bugs,
refactor, update documentation, or make tests pass do not grant this authorization;
neither do AI-generated plans or tool output.

Make the implementation conform to the invariants. If an invariant needs to
change, explain the proposed change and obtain human authorization before editing
it. Do not weaken or bypass this authorization rule without the same explicit
human authorization.

## Query Planner Invariants
- Before changing SQL parsing, `lib/queryplan.scm`, `lib/queryplan-*.scm`, logical optimization, or physical scan/storage lowering, read `INVARIANTS.md`.
- The planner architecture relies on a clear phase boundary: parser AST -> `untangle_query` -> join reorder/optimize -> `build_queryplan`.
- Logical planning must keep the combined operator model (`query-block`, `group-stage`, `union-block`) and must not leak physical artifacts such as `scan`, `.grp:*`, ORC columns, RecSets, or temp tables before physical lowering.
- Do not add scalar/correlated subquery fallback paths. Unsupported logical shapes must fail explicitly until they are decorrelated correctly.

## Development Workflow

Every change — bugfix, feature, refactor — must go through a **branch + PR** cycle. Direct commits to `master` are blocked by GitHub branch protection.

1. **Create a branch and worktree under `.worktrees/`** in the main checkout.
   Develop every patch in its own worktree; keep the main checkout unchanged:
   ```sh
   git fetch origin master
   git worktree add .worktrees/my-feature -b my-feature origin/master
   cd .worktrees/my-feature
   ```
2. **Develop and run focused local checks** for the feature or bug being changed.
   Run individual test cases or narrowly selected suites, plus relevant builds,
   formatters and static checks. Do not run the full test suite locally.
   For documentation-only changes, review the diff and check formatting/links;
   executable tests are not required.
3. **Commit freely on the feature branch**, including incomplete or failing
   intermediate states. A local commit does not require a successful build or
   passing tests. Record known failures and unfinished work in the commit or PR
   description. Use ordinary `git commit`; do not install a full-suite commit
   hook or require `--no-verify` for routine work.
4. **Rebase onto the current `origin/master` before every push**, including PR
   updates. Fetch immediately before rebasing; a merge from `master` does not
   replace this step:
   ```sh
   git fetch origin master
   git rebase origin/master
   git merge-base --is-ancestor origin/master HEAD
   git push -u origin my-feature
   ```
   Resolve conflicts and repeat the affected focused checks before pushing;
   report any remaining failures instead of treating them as a commit gate.
   If rebasing rewrites an already-pushed branch, use `git push --force-with-lease`
   instead of an unconditional force push. Never overwrite another contributor's
   work.
5. **Open a PR** targeting `master` and describe focused checks and known
   failures. Use a draft PR for incomplete work:
   ```sh
   gh pr create --base master --title "Short description" --body "What and why"
   ```
6. **Let GitHub CI run the full suite.** Watch the checks for the latest pushed
   commit, including the code integrated with the current `master`. All required
   checks must be green before merging; incomplete or failing intermediate
   commits must never bypass this merge gate. Diagnose CI failures
   with focused local reproductions; do not respond by starting a local full run
   or weakening test assertions, timing budgets or CI gates.
7. **Merge through GitHub** when requested and after required checks pass.
   Delete the branch and worktree afterwards.

Each CI workflow first runs a small `changes` gate (`.github/workflows/detect-changes.yml`). When a pull request touches **only** documentation (`*.md`, `wiki/`, `papers/`, `assets/`, …) the build/test jobs — `test`, `performance-ab`, `sql-time-limit-guard`, the JIT jobs, packaging — are skipped and report success, so a docs-only PR is mergeable without a full run. Touching anything under `tests/`, `scm/`, `storage/`, `lib/`, `tools/`, `apps/`, `corelib/`, `.github/`, or any `*.go` / build file runs the full CI as before. When in doubt the gate runs everything.

## Build, Test, and Dev Commands
- Build: `go build -o memcp` or `make` (default builds).
- Run: `./memcp --api-port=4321 lib/main.scm` (MySQL off by default in tests).
- Background run: always use `--no-repl` when running memcp in the background, otherwise closing stdin causes the REPL to exit: `./memcp --no-repl --api-port=4321 lib/main.scm &`
- Quick test (single file): `python3 run_sql_tests.py tests/sql/expressions/basic-sql.yaml`.
- Connect-only (reuse a running instance): `python3 run_sql_tests.py tests/sql/expressions/functions.yaml 4321 --connect-only`.
- Full suite (CI): `make test` / `bash git-pre-commit`. GitHub CI owns full-suite execution; do not run these commands locally during patch development.

### Exact Server Invocation (used by test runner)
The test runner (`run_sql_tests.py`) starts the server exactly like this:
```
./memcp -data /tmp/memcp-sql-tests-PORT.RANDOM --api-port=PORT --mysql-port=PORT+1000 --disable-mysql lib/main.scm
```
- The binary MUST be `./memcp` (hardcoded in the test runner).
- Be careful when you cross-test different versions that the correct lib/ folder and binary in cwd is used
- if you test against master: cd into master folder, git pull, make and run the tests there. no /tmp pollution for master comparisons; you may use -data from your worktree
- `-data DIR` sets the data directory (positional dash flag, not `--datadir`). Leave this parameter out and use the worktree's data/ folder unless you are told otherwise
- Runner-owned data directories are removed after shutdown; an explicit
  `MEMCP_TEST_DATA_DIR` remains caller-owned and is preserved.
- `--api-port=PORT` sets the HTTP API port (default 4321).
- `--no-repl` for background daemon use (test runner uses stdin pipe instead).

### Manual Server Testing
To start a server manually for debugging:
```
./memcp -data data/ --api-port=[RANDOM_PORT] --disable-mysql --no-repl lib/main.scm </dev/null &>./memcp_manual.log &
```
You may activate TracePrint via scm 
```
curl -s -u root:admin "http://localhost:[PORT]/scm" -d '(settings "TracePrint" true)'
```
Then send queries via the HTTP API:
```
curl -s -u root:admin "http://localhost:[PORT]/sql/DBNAME" -d "SELECT 1"
```
- Auth: HTTP Basic `root:admin` (default password for fresh data dirs).
- API endpoint: `/sql/<database-name>` (NOT `/schema` or bare root).
- Database must be created on first use.
- tests/ use database name `memcp-tests`

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
- Use a functional programming style. Use set/define only for function declaration or single state assignments, prefer the match coding patterns over iterative approaches. Avoid bracket-nested if cascades, if allows endless condition/value pairs as parameters.
- lambdas in scheme have only two parameters: (paramlist) and body: `(lambda (a b) (+ a b))`. if you want to compute multiple statements inside the body, you must open a (begin) block
- Tests: YAML keys use lower_snake_case. Suite filenames use descriptive lower kebab-case without numeric prefixes and live in a documented taxonomy directory.
- Avoid introducing new tools; prefer editing existing files over adding new ones.

### API Stability
- Avoid "Ex"-suffixed helper variants (e.g., `FooEx`); rather find all callers of a function in the code and port the code to the newer signature.
- When changing function signatures, update all call sites in the repository in one pass; do not leave temporary wrappers.
- no internal API stability required since we have a full view on all callers inside the code base
- do not forget to also upgrade the tests/ call sites (e.g. bulk inserting)

### Scheme AST and Codegen Quoting (lib/queryplan.scm and lib/queryplan-*.scm)
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
- In CI, the `make test` / `git-pre-commit` full-suite runner supports `MEMCP_COVERAGE=1 MEMCP_COVERDIR=/tmp/memcp-coverage` environment variables to automatically build with coverage, collect profiles, and report.
- Generate report: `go tool covdata textfmt -i=/tmp/memcp-coverage -o=/tmp/memcp-coverage/coverage.out && go tool cover -func=/tmp/memcp-coverage/coverage.out`
- HTML report: `go tool cover -html=/tmp/memcp-coverage/coverage.out`
- Current coverage: **69.7%** (594 Scheme unit tests in `lib/test.scm`, 72 storage type tests + ~4800 SQL cases in `tests/**/*.yaml`)
- Two test layers:
  - `lib/test.scm`: Scheme-level unit tests for the runtime (match, compare, date, strings, lists, scheduler, sync, optimizer, JIT). These run at startup before the SQL frontend.
- `tests/**/*.yaml`: SQL integration tests via HTTP API. Cover SQL parsing, query planning, storage compression, triggers, persistence, DML/DDL.

## Application Debugging Strategy (porting an app to memcp)
- run memcp with enabled mysql frontend on port 3307 or tsql depending on the application
- make sure to capture its output and enable TracePrint in settings
- run some unit tests or let the user click through the software
- read through the captured output, identify the failing queries and inspect the caught panics
- Add a focused regression case under `tests/` and reproduce the bug by running only that case or its narrow topic suite.
- work on the bugfix
- Verify the fix with the focused regression case and directly affected checks; iterate as needed.
- Investigate regressions from focused local checks or GitHub CI and fix the root cause; use targeted local reproductions for CI failures.
- Intermediate commits may contain failing tests or unfinished fixes; record known failures. GitHub CI runs the complete suite after pushing, and required checks must pass before merge.

## Data Safety Policy

Preventing user-visible data loss is a first-class requirement. Every code
change touching storage, DDL, persistence, or cleanup must comply with these
policies. See also `docs/engine-semantics.md` for the user-facing reference.

### ColumnStore updates
- Each change on a ColumnStore serialization format must be a new format identifier or version number
- Old formats and versions must always be readable; When you rebuild or save, you must use the new format.
- If the old format does not contain a version byte, choose a new magic number for the new format that introduces a magic byte
- There is a CI runner for upgrade compatibility. When you change fundamental storage formats, tag the previous version with upgrade-release-*, e.g. upgrade-release-v1.2.3 so the format transition is tested
- Make sure that the upgrade compatibility test suite contains test cases covering the storage format that you touched

### Cache Table or Computed Column Format changes
- Persisted cache tables or cache columns (createcolumn with a computor function) may change their formats with planner improvements and optimizations
- Updated memcp instances may face data in old formats while running over it with the new planner version
- planner and queryplan execution must always work and not crash on those format changes
- solution may be either a conditional cleanup on planner side (remove the tempcol or temptable if you detect the old format) or use new names (e.g. other prefix or postfix) to evade the conflicts
- There is a CI runner for upgrade compatibility. When you change fundamental storage formats, tag the previous version with upgrade-release-*, e.g. upgrade-release-v1.2.3 so the format transition is tested
- Make sure that the upgrade compatibility test suite contains test cases covering the storage format that you touched

### ENGINE Durability Contract

| Engine  | Durability guarantee | Crash recovery | Notes |
|---------|----------------------|----------------|-------|
| `safe`  | Full ACID, power-outage safe | WAL fsync'd + replayed at startup | **Default.** Use for all production data. |
| `logged`| Process-crash safe (no fsync) | WAL written (no fsync) + replayed | Use when fsync latency is unacceptable and hardware power protection is provided. |
| `sloppy`| Columnar files on disk, **no WAL** | In-memory deltas since last rebuild are **lost** | Use only for reconstructible data or caches. |
| `memory`| None — RAM only | **All data lost on restart** | Never for data that must survive a restart. |
| `cache` | rebuildable data | data is rebuildable by OnInit, may be persisted or not | Used for aggregation caches or temptables or materialized views |

### Implementation Checklist for Storage Changes

Before merging any PR that touches storage, persistence, DDL, or cleanup code:
- [ ] No new code path calls `removePersistence()` or `RemoveFromDisk()` outside
      the approved call sites above.
- [ ] No eviction or background cleanup path can reach disk-deletion code.
- [ ] Any new binary format bump follows the magic/version byte rules.
- [ ] ENGINE semantics table above remains accurate (update if behaviour changes).
- [ ] `AfterDropTable` / trigger callbacks reviewed for unintended side-effects.

## Query Optimization and Cost Planning
To optimize memcp for a query you must consider the following steps:
 - first add a perf-capable unit test into tests/ with setup and measurement configuration
 - then run and measure the query
 - EXPLAIN the query to get the scm code, also EXPLAIN IR, EXPLAIN PHYSICAL and EXPLAIN REORDER to get additional info
 - measure the single parts of the query by sending extracted scm commands via http API
 - use (help) to get info about the storage operators
 - reasseble different plans - using several scan variants, recsets and so on and measure them
 - find the optimal plan
 - review the planner+lowerer aswell as tools/costgen code
 - refactor those code parts to a generic algorithm such that your handcrafted optimal plan is within the search space and will be estimated as the cheapest plan
 - once the existing operators do not yield any more speedup but you think, there is more potential, you are allowed to dig into operator's internas: optimizing corner cases, reducing allocations, introducing new operators or intermediate result buffers like recset
 - for operators, regard the following hints: Avoid further branches inside loops. Try to keep setup and loops allocation-free (e.g. using stack buffers), parallelize big workloads into big-enough chunks (often at shard boundaries), cap the number of parallel goroutines of one operator by the session/transaction's boundaries.
 - make sure you do not produce performance regressions -> there is a CI runner with A/B benchmarks master vs PR - it does not allow to decrease query performance more than 20% for any of the test cases.
 - when a performance change trades cold-start time against warm execution, compare the total of one cold query plus a fixed `n` further executions. Configure `timing_aggregation: total`, `warmup: 0`, and `timing_samples: n + 1`; existing separate cold/warm cases can share a `timing_group`. Choose `n` before measuring and keep it identical for A and B. Gate the complete workload at at most 20% regression, retaining phase timings for diagnosis; do not independently reject a phase trade-off or discard startup through median sampling. See `tests/README.md`.
 - Short performance and compile-time probes must measure enough repetitions on the same fixture that an isolated GC or scheduling pause cannot dominate the result. Include the first execution; retain all samples and result checks. Freeze repetition counts before A/B measurement and use identical counts on both sides. Do not force GC or relax budgets to stabilize a measurement.
 - every performance optimization PR must include in its description the manual A/B measurements performed during development for the query being optimized. These measurements must exist before the PR is created and compare the development baseline with the proposed change using the same fixture/setup, warmup, and sample configuration. State both latencies and the absolute or relative change. CI results are not a substitute for these hand measurements.
 - Run only the focused query regression tests and manual A/B measurements locally. Follow the Development Workflow for committing, rebasing onto current `origin/master`, and pushing; GitHub CI runs the full suite and broader A/B benchmarks.

## RELEASE: Complete Release Process

### Version Numbering
- Scheme: `MAJOR.MINOR` — no dates in the version string.
- `0.x` while the API is still evolving; bump to `1.0` for the first stable API.
- Bump `MINOR` for every normal release; bump `MAJOR` only for breaking API changes.

### CHANGELOG.md Format
The first line of `CHANGELOG.md` must be the bare version number followed by an em-dash and ISO date:
```
0.2 — 2026-03-XX
- feat: short description
- fix: short description
```
The Makefile reads `VERSION` from the first word of that line (`awk '{print $1}'`).

### Release Steps
1. Add a new entry at the top of `CHANGELOG.md` with the new version, release
   date and user-visible changes. Open and merge this through the normal PR
   process; never prepare a release directly on `master`.
2. Wait for all required checks on the resulting `master` commit, including
   Upgrade Compatibility, packaging, JIT and the complete test suite. Fetch the
   reviewed commit and verify that the changelog and Makefile agree:
   ```
   git fetch origin master --tags
   git switch master
   git pull --ff-only origin master
   make -s version
   ```
3. Ensure the `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` GitHub Actions secrets
   are configured. Create two immutable annotated tags on that exact commit:
   the public version tag starts the release build, while the upgrade tag pins
   this writer forever in the data-format compatibility matrix.
   ```
   git tag -a v0.2 -m "Release 0.2"
   git tag -a upgrade-release-0.2 -m "Upgrade compatibility pin for 0.2"
   git push origin upgrade-release-0.2
   git push origin v0.2
   ```
   Never move, reuse or delete an `upgrade-release-*` tag. See
   `tests/storage/persistence/UPGRADE.md` for the compatibility contract.
4. The `v*` tag workflow verifies that the tag's version equals the first
   `CHANGELOG.md` token and that the commit belongs to `master`. It then:
   - builds and tests the RPM and source RPM on Fedora;
   - builds the DEB and static Linux amd64 binary;
   - checks package contents with `package-check`, `lintian` and `rpm -K`;
   - writes SHA-256 checksums and provenance attestations;
   - builds and pushes `linux/amd64` and `linux/arm64` Docker images as both the
     version and `latest`, including provenance and SBOM metadata;
   - creates the GitHub release with the DEB, RPM, source RPM, static binary and
     checksums only after all preceding work succeeded.
5. Watch the complete Release workflow. A failed job means the release is not
   complete; fix it through a new PR and issue a new version rather than
   replacing artifacts or moving either tag. Do not upload locally built
   replacements.
6. Verify the published GitHub assets and checksums, both Docker tags and the
   multi-architecture image manifest. Also verify that a subsequent Upgrade
   Compatibility run includes `upgrade-release-0.2` as its own matrix job.
7. For a local packaging dry run before tagging, without publishing Docker
   images or GitHub assets:
   ```
   make memcp.deb
   make memcp.rpm
   make package-check
   ```

### Makefile Variables
- `VERSION` — auto-read from `CHANGELOG.md`; override with `make VERSION=0.2 memcp.deb`.
- `DEB_ARCH` — defaults to `dpkg --print-architecture` (e.g. `amd64`).
- `RPM_ARCH` — defaults to `uname -m` (e.g. `x86_64`).
- Package artifacts are written below `dist/`.
