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
